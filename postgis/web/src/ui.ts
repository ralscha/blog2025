import maplibregl, { GeoJSONSource, LngLatBounds } from 'maplibre-gl';

type Mode = 'nearby' | 'clusters' | 'corridor' | 'geofence';
type PointCoord = [number, number];

interface NearbyStore {
  storeNumber: string;
  countryCode: string;
  city: string;
  streetAddress: string;
  latitude: number;
  longitude: number;
  distanceMeters: number;
}

interface ClusterResult {
  clusterID: number;
  storeCount: number;
  centerLatitude: number;
  centerLongitude: number;
}

interface CorridorStore {
  storeNumber: string;
  countryCode: string;
  city: string;
  streetAddress: string;
  latitude: number;
  longitude: number;
  distanceToRouteMeters: number;
}

interface CountrySummary {
  countryCode: string;
  storeCount: number;
}

interface CountriesResponse {
  countries: CountrySummary[];
}

interface NearbyResponse {
  stores: NearbyStore[];
}

interface ClustersResponse {
  countryCode: string;
  clusters: ClusterResult[];
}

interface CorridorResponse {
  stores: CorridorStore[];
}

interface TruckGeofenceStatus {
  truckID: string;
  driverName: string;
  geofenceID: string;
  geofenceName: string;
  latitude: number;
  longitude: number;
  updatedAt: string;
  inside: boolean;
}

interface TruckGeofenceEvent {
  id: number;
  truckID: string;
  driverName: string;
  geofenceID: string;
  geofenceName: string;
  eventType: 'entered' | 'exited';
  latitude: number;
  longitude: number;
  occurredAt: string;
}

interface GeofenceArea {
  geofenceID: string;
  name: string;
  category: string;
  geometry: GeoJSON.Polygon;
}

interface GeofencesResponse {
  geofences: GeofenceArea[];
}

interface GeofenceLiveResponse {
  geofence: GeofenceArea;
  trucks: TruckGeofenceStatus[];
  events: TruckGeofenceEvent[];
  serverTime: string;
}

type FeatureCollection = GeoJSON.FeatureCollection<GeoJSON.Geometry>;

const defaultGeofenceID = 'home-depot-sodo';
const defaultGeofenceRefreshMs = 2000;

const emptyFeatureCollection: FeatureCollection = {
  type: 'FeatureCollection',
  features: [],
};

export class DemoApp {
  private readonly root: HTMLDivElement;

  private readonly countries: CountrySummary[] = [];

  private readonly state = {
    mode: 'nearby' as Mode,
    geofenceRefreshMs: defaultGeofenceRefreshMs,
  };

  private map?: maplibregl.Map;
  private resultsElement?: HTMLDivElement;
  private statusElement?: HTMLParagraphElement;
  private geofencePollTimer?: number;

  constructor(root: HTMLDivElement) {
    this.root = root;
  }

  mount(): void {
    this.root.innerHTML = this.template();
    this.resultsElement =
      this.root.querySelector<HTMLDivElement>('[data-results]') ?? undefined;
    this.statusElement =
      this.root.querySelector<HTMLParagraphElement>('[data-status]') ??
      undefined;

    this.bindTabs();
    this.bindForms();
    this.createMap();
    void this.bootstrap();
  }

  private async bootstrap(): Promise<void> {
    this.setStatus(
      'Loading country list and default nearby search...',
      'normal',
    );
    await this.loadCountries();
    await this.runNearby();
  }

  private template(): string {
    return `
      <div class="shell">
        <aside class="panel">
          <h1>Starbucks Geo Atlas</h1>
          <p class="intro">Four spatial demos on one map: nearest-neighbor search, K-means clustering, route corridor discovery, and a live truck geofence feed.</p>

          <div class="mode-switch" role="tablist" aria-label="Spatial demos">
            <button class="mode-button is-active" data-mode-button="nearby">Nearby</button>
            <button class="mode-button" data-mode-button="clusters">Clusters</button>
            <button class="mode-button" data-mode-button="corridor">Corridor</button>
            <button class="mode-button" data-mode-button="geofence">Geofence</button>
          </div>

          <section class="mode-panel is-visible" data-mode-panel="nearby">
            <form data-form="nearby" class="form-grid">
              <label><span>Latitude</span><input name="lat" type="number" step="0.000001" value="35.6895" /></label>
              <label><span>Longitude</span><input name="lon" type="number" step="0.000001" value="139.6917" /></label>
              <label><span>Radius meters</span><input name="radius" type="number" step="100" value="1500" /></label>
              <label><span>Limit</span><input name="limit" type="number" min="1" value="12" /></label>
              <button class="action-button" type="submit">Run nearby search</button>
            </form>
          </section>

          <section class="mode-panel" data-mode-panel="clusters">
            <form data-form="clusters" class="form-grid">
              <label><span>Country</span><select name="country" data-country-select></select></label>
              <label><span>Cluster count</span><input name="k" type="number" min="2" max="12" value="5" /></label>
              <button class="action-button" type="submit">Build clusters</button>
            </form>
          </section>

          <section class="mode-panel" data-mode-panel="corridor">
            <form data-form="corridor" class="form-grid corridor-grid">
              <label><span>From lat</span><input name="fromLat" type="number" step="0.000001" value="35.6585" /></label>
              <label><span>From lon</span><input name="fromLon" type="number" step="0.000001" value="139.7013" /></label>
              <label><span>To lat</span><input name="toLat" type="number" step="0.000001" value="35.6895" /></label>
              <label><span>To lon</span><input name="toLon" type="number" step="0.000001" value="139.6917" /></label>
              <label><span>Corridor width</span><input name="distance" type="number" step="50" value="600" /></label>
              <label><span>Limit</span><input name="limit" type="number" min="1" value="15" /></label>
              <button class="action-button" type="submit">Find route stores</button>
            </form>
          </section>

          <section class="mode-panel" data-mode-panel="geofence">
            <form data-form="geofence" class="form-grid corridor-grid">
              <label><span>Geofence id</span><input name="geofence" type="text" value="${defaultGeofenceID}" /></label>
              <label><span>Event limit</span><input name="eventLimit" type="number" min="1" max="40" value="12" /></label>
              <label><span>Refresh ms</span><input name="refreshMs" type="number" min="500" step="250" value="${defaultGeofenceRefreshMs}" /></label>
              <button class="action-button" type="submit">Start live geofence view</button>
            </form>
            <p class="panel-note">This view polls the Go API for the Home Depot polygon, latest truck positions, and recent enter or exit events.</p>
          </section>

          <p class="status" data-status></p>
          <div class="results" data-results></div>
        </aside>

        <main class="map-frame">
          <div id="map" class="map"></div>
        </main>
      </div>
    `;
  }

  private bindTabs(): void {
    const buttons =
      this.root.querySelectorAll<HTMLButtonElement>('[data-mode-button]');
    const panels = this.root.querySelectorAll<HTMLElement>('[data-mode-panel]');

    buttons.forEach((button) => {
      button.addEventListener('click', () => {
        const nextMode = button.dataset.modeButton as Mode;
        this.state.mode = nextMode;

        buttons.forEach((entry) =>
          entry.classList.toggle('is-active', entry === button),
        );
        panels.forEach((panel) =>
          panel.classList.toggle(
            'is-visible',
            panel.dataset.modePanel === nextMode,
          ),
        );

        if (nextMode === 'geofence') {
          void this.startGeofencePolling();
        } else {
          this.stopGeofencePolling();
        }
      });
    });
  }

  private bindForms(): void {
    const nearbyForm = this.root.querySelector<HTMLFormElement>(
      '[data-form="nearby"]',
    );
    const clustersForm = this.root.querySelector<HTMLFormElement>(
      '[data-form="clusters"]',
    );
    const corridorForm = this.root.querySelector<HTMLFormElement>(
      '[data-form="corridor"]',
    );
    const geofenceForm = this.root.querySelector<HTMLFormElement>(
      '[data-form="geofence"]',
    );

    nearbyForm?.addEventListener('submit', (event) => {
      event.preventDefault();
      void this.runNearby();
    });

    clustersForm?.addEventListener('submit', (event) => {
      event.preventDefault();
      void this.runClusters();
    });

    corridorForm?.addEventListener('submit', (event) => {
      event.preventDefault();
      void this.runCorridor();
    });

    geofenceForm?.addEventListener('submit', (event) => {
      event.preventDefault();
      this.state.mode = 'geofence';
      this.root
        .querySelectorAll<HTMLButtonElement>('[data-mode-button]')
        .forEach((button) => {
          button.classList.toggle(
            'is-active',
            button.dataset.modeButton === 'geofence',
          );
        });
      this.root
        .querySelectorAll<HTMLElement>('[data-mode-panel]')
        .forEach((panel) => {
          panel.classList.toggle(
            'is-visible',
            panel.dataset.modePanel === 'geofence',
          );
        });
      void this.startGeofencePolling();
    });
  }

  private createMap(): void {
    this.map = new maplibregl.Map({
      container: 'map',
      style: 'https://demotiles.maplibre.org/style.json',
      center: [139.6917, 35.6895],
      zoom: 12,
    });

    this.map.addControl(
      new maplibregl.NavigationControl({ visualizePitch: true }),
      'top-right',
    );

    this.map.on('load', () => {
      if (!this.map) {
        return;
      }
      this.map.setGlyphs(
        'https://fonts.openmaptiles.org/{fontstack}/{range}.pbf',
      );

      this.map.addSource('geofences', {
        type: 'geojson',
        data: emptyFeatureCollection,
      });
      this.map.addSource('results', {
        type: 'geojson',
        data: emptyFeatureCollection,
      });
      this.map.addSource('route', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] },
      });
      this.map.addSource('geofence', {
        type: 'geojson',
        data: emptyFeatureCollection,
      });

      this.map.addLayer({
        id: 'geofences-fill',
        type: 'fill',
        source: 'geofences',
        paint: {
          'fill-color': '#22c55e',
          'fill-opacity': 0.09,
        },
      });

      this.map.addLayer({
        id: 'geofences-outline',
        type: 'line',
        source: 'geofences',
        paint: {
          'line-color': '#4ade80',
          'line-width': 2,
          'line-opacity': 0.55,
          'line-dasharray': [4, 3],
        },
      });

      this.map.addLayer({
        id: 'geofence-fill',
        type: 'fill',
        source: 'geofence',
        paint: {
          'fill-color': '#22c55e',
          'fill-opacity': 0.18,
        },
      });

      this.map.addLayer({
        id: 'geofence-outline',
        type: 'line',
        source: 'geofence',
        paint: {
          'line-color': '#86efac',
          'line-width': 3,
          'line-opacity': 0.9,
        },
      });

      this.map.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        paint: {
          'line-color': '#ff6b35',
          'line-width': 4,
          'line-opacity': 0.9,
        },
      });

      this.map.addLayer({
        id: 'geofences-label',
        type: 'symbol',
        source: 'geofences',
        layout: {
          'text-field': ['get', 'name'],
          'text-font': ['Open Sans Regular'],
          'text-size': 12,
          'text-anchor': 'center',
        },
        paint: {
          'text-color': '#86efac',
          'text-halo-color': '#10212b',
          'text-halo-width': 1.5,
        },
      });

      this.map.addLayer({
        id: 'results-circles',
        type: 'circle',
        source: 'results',
        paint: {
          'circle-color': [
            'case',
            ['==', ['get', 'kind'], 'cluster'],
            '#0d9488',
            ['==', ['get', 'kind'], 'truck-inside'],
            '#22c55e',
            ['==', ['get', 'kind'], 'truck-outside'],
            '#fbbf24',
            '#f43f5e',
          ],
          'circle-radius': [
            'case',
            ['==', ['get', 'kind'], 'cluster'],
            [
              'interpolate',
              ['linear'],
              ['coalesce', ['get', 'storeCount'], 0],
              1,
              12,
              50,
              26,
              500,
              40,
            ],
            [
              'any',
              ['==', ['get', 'kind'], 'truck-inside'],
              ['==', ['get', 'kind'], 'truck-outside'],
            ],
            10,
            8,
          ],
          'circle-stroke-color': '#fff8e8',
          'circle-stroke-width': 2,
          'circle-opacity': 0.92,
        },
      });

      this.map.addLayer({
        id: 'results-labels',
        type: 'symbol',
        source: 'results',
        layout: {
          'text-field': ['coalesce', ['get', 'label'], ['get', 'storeNumber']],
          'text-font': ['Open Sans Regular'],
          'text-size': 11,
          'text-offset': [0, 1.2],
        },
        paint: {
          'text-color': '#fff8e8',
          'text-halo-color': '#10212b',
          'text-halo-width': 1,
        },
      });

      void this.loadAllGeofences();
    });
  }

  private async loadAllGeofences(): Promise<void> {
    const payload = await this.fetchJSON<GeofencesResponse>('/api/geofences');
    const features = payload.geofences.map((g) => ({
      type: 'Feature' as const,
      geometry: g.geometry,
      properties: {
        geofenceID: g.geofenceID,
        name: g.name,
        category: g.category,
      },
    }));
    this.setGeoJSON('geofences', { type: 'FeatureCollection', features });
  }

  private async loadCountries(): Promise<void> {
    const payload = await this.fetchJSON<CountriesResponse>(
      '/api/countries?limit=40',
    );
    this.countries.splice(0, this.countries.length, ...payload.countries);

    const select = this.root.querySelector<HTMLSelectElement>(
      '[data-country-select]',
    );
    if (!select) {
      return;
    }

    select.innerHTML = this.countries
      .map((country, index) => {
        const selected =
          index === 0 || country.countryCode === 'JP' ? 'selected' : '';
        return `<option value="${country.countryCode}" ${selected}>${country.countryCode} (${country.storeCount})</option>`;
      })
      .join('');
  }

  private async runNearby(): Promise<void> {
    const form = this.root.querySelector<HTMLFormElement>(
      '[data-form="nearby"]',
    );
    if (!form) {
      return;
    }

    const formData = new FormData(form);
    const query = new URLSearchParams({
      lat: String(formData.get('lat') ?? ''),
      lon: String(formData.get('lon') ?? ''),
      radius: String(formData.get('radius') ?? ''),
      limit: String(formData.get('limit') ?? ''),
    });

    this.setStatus('Running nearest-neighbor query...', 'normal');
    const payload = await this.fetchJSON<NearbyResponse>(
      `/api/nearby?${query.toString()}`,
    );
    this.renderNearby(payload.stores);
    this.drawNearby(payload.stores);
    this.setStatus(
      `Rendered ${payload.stores.length} nearby stores.`,
      'success',
    );
  }

  private async runClusters(): Promise<void> {
    const form = this.root.querySelector<HTMLFormElement>(
      '[data-form="clusters"]',
    );
    if (!form) {
      return;
    }

    const formData = new FormData(form);
    const query = new URLSearchParams({
      country: String(formData.get('country') ?? 'JP'),
      k: String(formData.get('k') ?? '5'),
    });

    this.setStatus('Computing K-means clusters...', 'normal');
    const payload = await this.fetchJSON<ClustersResponse>(
      `/api/clusters?${query.toString()}`,
    );
    this.renderClusters(payload.countryCode, payload.clusters);
    this.drawClusters(payload.clusters);
    this.setStatus(
      `Rendered ${payload.clusters.length} cluster centers for ${payload.countryCode}.`,
      'success',
    );
  }

  private async runCorridor(): Promise<void> {
    const form = this.root.querySelector<HTMLFormElement>(
      '[data-form="corridor"]',
    );
    if (!form) {
      return;
    }

    const formData = new FormData(form);
    const query = new URLSearchParams({
      fromLat: String(formData.get('fromLat') ?? ''),
      fromLon: String(formData.get('fromLon') ?? ''),
      toLat: String(formData.get('toLat') ?? ''),
      toLon: String(formData.get('toLon') ?? ''),
      distance: String(formData.get('distance') ?? ''),
      limit: String(formData.get('limit') ?? ''),
    });

    this.setStatus('Searching for stores along the route...', 'normal');
    const payload = await this.fetchJSON<CorridorResponse>(
      `/api/corridor?${query.toString()}`,
    );
    this.renderCorridor(payload.stores);
    this.drawCorridor(payload.stores, {
      fromLat: Number(formData.get('fromLat')),
      fromLon: Number(formData.get('fromLon')),
      toLat: Number(formData.get('toLat')),
      toLon: Number(formData.get('toLon')),
    });
    this.setStatus(
      `Rendered ${payload.stores.length} route-adjacent stores.`,
      'success',
    );
  }

  private async startGeofencePolling(): Promise<void> {
    const form = this.root.querySelector<HTMLFormElement>(
      '[data-form="geofence"]',
    );
    if (!form) {
      return;
    }

    const formData = new FormData(form);
    const refreshValue = Number(
      formData.get('refreshMs') ?? defaultGeofenceRefreshMs,
    );
    this.state.geofenceRefreshMs =
      Number.isFinite(refreshValue) && refreshValue >= 500
        ? refreshValue
        : defaultGeofenceRefreshMs;

    this.stopGeofencePolling();
    await this.runGeofenceSnapshot();
    this.geofencePollTimer = window.setInterval(() => {
      void this.runGeofenceSnapshot(false);
    }, this.state.geofenceRefreshMs);
  }

  private stopGeofencePolling(): void {
    if (this.geofencePollTimer !== undefined) {
      window.clearInterval(this.geofencePollTimer);
      this.geofencePollTimer = undefined;
    }
  }

  private async runGeofenceSnapshot(updateStatus = true): Promise<void> {
    const form = this.root.querySelector<HTMLFormElement>(
      '[data-form="geofence"]',
    );
    if (!form) {
      return;
    }

    const formData = new FormData(form);
    const geofence =
      String(formData.get('geofence') ?? defaultGeofenceID).trim() ||
      defaultGeofenceID;
    const eventLimit = String(formData.get('eventLimit') ?? '12');
    const query = new URLSearchParams({ geofence, eventLimit });

    if (updateStatus) {
      this.setStatus(
        'Refreshing geofence positions and recent events...',
        'normal',
      );
    }

    const payload = await this.fetchJSON<GeofenceLiveResponse>(
      `/api/geofence/live?${query.toString()}`,
    );
    this.renderGeofence(payload);
    this.drawGeofence(payload);

    const insideCount = payload.trucks.filter((truck) => truck.inside).length;
    this.setStatus(
      `${payload.geofence.name}: ${insideCount} trucks currently inside, ${payload.events.length} recent events loaded.`,
      'success',
    );
  }

  private drawNearby(stores: NearbyStore[]): void {
    const features = stores.map((store) => ({
      type: 'Feature' as const,
      geometry: {
        type: 'Point' as const,
        coordinates: [store.longitude, store.latitude],
      },
      properties: {
        kind: 'store',
        storeNumber: store.storeNumber,
        label: `${store.distanceMeters} m`,
      },
    }));

    this.setGeoJSON('results', { type: 'FeatureCollection', features });
    this.setGeoJSON('route', emptyFeatureCollection);
    this.setGeoJSON('geofence', emptyFeatureCollection);
    this.fitPoints(
      stores.map((store) => pointCoord(store.longitude, store.latitude)),
    );
  }

  private drawClusters(clusters: ClusterResult[]): void {
    const features = clusters.map((cluster) => ({
      type: 'Feature' as const,
      geometry: {
        type: 'Point' as const,
        coordinates: [cluster.centerLongitude, cluster.centerLatitude],
      },
      properties: {
        kind: 'cluster',
        clusterID: cluster.clusterID,
        storeCount: cluster.storeCount,
        label: `${cluster.storeCount}`,
      },
    }));

    this.setGeoJSON('results', { type: 'FeatureCollection', features });
    this.setGeoJSON('route', emptyFeatureCollection);
    this.setGeoJSON('geofence', emptyFeatureCollection);
    this.fitPoints(
      clusters.map((cluster) =>
        pointCoord(cluster.centerLongitude, cluster.centerLatitude),
      ),
    );
  }

  private drawCorridor(
    stores: CorridorStore[],
    route: { fromLat: number; fromLon: number; toLat: number; toLon: number },
  ): void {
    const features = stores.map((store) => ({
      type: 'Feature' as const,
      geometry: {
        type: 'Point' as const,
        coordinates: [store.longitude, store.latitude],
      },
      properties: {
        kind: 'store',
        storeNumber: store.storeNumber,
        label: `${store.distanceToRouteMeters} m`,
      },
    }));

    this.setGeoJSON('results', { type: 'FeatureCollection', features });
    this.setGeoJSON('geofence', emptyFeatureCollection);
    this.setGeoJSON('route', {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          geometry: {
            type: 'LineString',
            coordinates: [
              [route.fromLon, route.fromLat],
              [route.toLon, route.toLat],
            ],
          },
          properties: {},
        },
      ],
    });

    this.fitPoints([
      ...stores.map((store) => pointCoord(store.longitude, store.latitude)),
      pointCoord(route.fromLon, route.fromLat),
      pointCoord(route.toLon, route.toLat),
    ]);
  }

  private drawGeofence(payload: GeofenceLiveResponse): void {
    const truckFeatures = payload.trucks.map((truck) => ({
      type: 'Feature' as const,
      geometry: {
        type: 'Point' as const,
        coordinates: [truck.longitude, truck.latitude],
      },
      properties: {
        kind: truck.inside ? 'truck-inside' : 'truck-outside',
        truckID: truck.truckID,
        label: truck.driverName,
      },
    }));

    const geofenceFeature = {
      type: 'Feature' as const,
      geometry: payload.geofence.geometry,
      properties: {
        geofenceID: payload.geofence.geofenceID,
        label: payload.geofence.name,
      },
    };

    this.setGeoJSON('results', {
      type: 'FeatureCollection',
      features: truckFeatures,
    });
    this.setGeoJSON('route', emptyFeatureCollection);
    this.setGeoJSON('geofence', {
      type: 'FeatureCollection',
      features: [geofenceFeature],
    });

    this.fitPoints([
      ...payload.trucks.map((truck) =>
        pointCoord(truck.longitude, truck.latitude),
      ),
      ...geometryPoints(payload.geofence.geometry),
    ]);
  }

  private setGeoJSON(
    sourceName: 'results' | 'route' | 'geofence' | 'geofences',
    data: FeatureCollection,
  ): void {
    if (!this.map || !this.map.isStyleLoaded()) {
      return;
    }

    const source = this.map.getSource(sourceName) as GeoJSONSource | undefined;
    source?.setData(data);
  }

  private fitPoints(points: PointCoord[]): void {
    if (!this.map || points.length === 0) {
      return;
    }

    const valid = points.filter(
      ([lng, lat]) => Number.isFinite(lng) && Number.isFinite(lat),
    );
    if (valid.length === 0) {
      return;
    }

    const bounds = valid.reduce(
      (accumulator, point) => accumulator.extend(point),
      new LngLatBounds(valid[0], valid[0]),
    );

    this.map.fitBounds(bounds, {
      padding: 64,
      maxZoom: 13,
      duration: 800,
    });
  }

  private renderNearby(stores: NearbyStore[]): void {
    if (!this.resultsElement) {
      return;
    }

    this.resultsElement.innerHTML = stores.length
      ? `<h2>Nearby results</h2>${stores
          .map(
            (store) => `
              <article class="result-card">
                <div class="result-title">${store.storeNumber}</div>
                <div class="result-meta">${this.placeLabel(store.city, store.streetAddress)}, ${store.countryCode}</div>
                <div class="result-metric">${store.distanceMeters} meters away</div>
              </article>
            `,
          )
          .join('')}`
      : `<h2>Nearby results</h2><p class="empty-state">No stores matched the current radius.</p>`;
  }

  private renderClusters(countryCode: string, clusters: ClusterResult[]): void {
    if (!this.resultsElement) {
      return;
    }

    this.resultsElement.innerHTML = clusters.length
      ? `<h2>Cluster centers for ${countryCode}</h2>${clusters
          .map(
            (cluster) => `
              <article class="result-card">
                <div class="result-title">Cluster ${cluster.clusterID}</div>
                <div class="result-meta">Center ${cluster.centerLatitude.toFixed(5)}, ${cluster.centerLongitude.toFixed(5)}</div>
                <div class="result-metric">${cluster.storeCount} stores</div>
              </article>
            `,
          )
          .join('')}`
      : `<h2>Clusters</h2><p class="empty-state">No stores were available for that country code.</p>`;
  }

  private renderCorridor(stores: CorridorStore[]): void {
    if (!this.resultsElement) {
      return;
    }

    this.resultsElement.innerHTML = stores.length
      ? `<h2>Route corridor matches</h2>${stores
          .map(
            (store) => `
              <article class="result-card">
                <div class="result-title">${store.storeNumber}</div>
                <div class="result-meta">${this.placeLabel(store.city, store.streetAddress)}, ${store.countryCode}</div>
                <div class="result-metric">${store.distanceToRouteMeters} meters from route</div>
              </article>
            `,
          )
          .join('')}`
      : `<h2>Route corridor matches</h2><p class="empty-state">No stores fell inside the current route corridor.</p>`;
  }

  private renderGeofence(payload: GeofenceLiveResponse): void {
    if (!this.resultsElement) {
      return;
    }

    const insideTrucks = payload.trucks.filter((truck) => truck.inside);
    const outsideTrucks = payload.trucks.filter((truck) => !truck.inside);

    this.resultsElement.innerHTML = `
      <h2>${payload.geofence.name}</h2>
      <article class="result-card result-card-accent">
        <div class="result-title">${insideTrucks.length} trucks currently inside the geofence</div>
        <div class="result-meta">Category: ${payload.geofence.category}. Last API snapshot ${this.formatTime(payload.serverTime)}.</div>
        <div class="result-metric">Tracking ${payload.trucks.length} fleet positions live</div>
      </article>
      <article class="result-card">
        <div class="result-title">Inside now</div>
        ${insideTrucks.length ? insideTrucks.map((truck) => this.truckCard(truck)).join('') : '<p class="empty-state">No trucks are inside the zone right now.</p>'}
      </article>
      <article class="result-card">
        <div class="result-title">Outside now</div>
        ${
          outsideTrucks.length
            ? outsideTrucks
                .slice(0, 6)
                .map((truck) => this.truckCard(truck))
                .join('')
            : '<p class="empty-state">All tracked trucks are currently inside the zone.</p>'
        }
      </article>
      <article class="result-card">
        <div class="result-title">Recent enter and exit events</div>
        ${payload.events.length ? payload.events.map((event) => this.eventCard(event)).join('') : '<p class="empty-state">No geofence events have been recorded yet.</p>'}
      </article>
    `;
  }

  private placeLabel(city: string, streetAddress: string): string {
    return city || streetAddress || 'Unknown location';
  }

  private truckCard(truck: TruckGeofenceStatus): string {
    const badgeClass = truck.inside ? 'is-entered' : 'is-exited';
    const badgeLabel = truck.inside ? 'Inside' : 'Outside';
    return `
      <div class="live-row">
        <div>
          <div class="result-title">${truck.driverName} <span class="result-inline-id">${truck.truckID}</span></div>
          <div class="result-meta">${truck.latitude.toFixed(5)}, ${truck.longitude.toFixed(5)} at ${this.formatTime(truck.updatedAt)}</div>
        </div>
        <span class="event-badge ${badgeClass}">${badgeLabel}</span>
      </div>
    `;
  }

  private eventCard(event: TruckGeofenceEvent): string {
    const badgeClass =
      event.eventType === 'entered' ? 'is-entered' : 'is-exited';
    return `
      <div class="live-row">
        <div>
          <div class="result-title">${event.driverName} <span class="result-inline-id">${event.truckID}</span></div>
          <div class="result-meta">${event.eventType} ${event.geofenceName} at ${this.formatTime(event.occurredAt)}</div>
        </div>
        <span class="event-badge ${badgeClass}">${event.eventType}</span>
      </div>
    `;
  }

  private formatTime(value: string): string {
    return new Date(value).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  }

  private async fetchJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);

    if (!response.ok) {
      let message = `Request failed with ${response.status}`;
      try {
        const payload = (await response.json()) as { error?: string };
        if (payload.error) {
          message = payload.error;
        }
      } catch {
        // Ignore JSON parsing errors and fall back to the HTTP status.
      }
      this.setStatus(message, 'error');
      throw new Error(message);
    }

    return (await response.json()) as T;
  }

  private setStatus(
    message: string,
    tone: 'normal' | 'success' | 'error',
  ): void {
    if (!this.statusElement) {
      return;
    }

    this.statusElement.textContent = message;
    this.statusElement.dataset.tone = tone;
  }
}

function pointCoord(longitude: number, latitude: number): PointCoord {
  return [longitude, latitude];
}

function geometryPoints(geometry: GeoJSON.Polygon): PointCoord[] {
  return geometry.coordinates.flatMap((ring) =>
    ring.map(([longitude, latitude]) => pointCoord(longitude, latitude)),
  );
}
