import { DecimalPipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { afterNextRender, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import type { FeatureCollection, LineString, Point } from 'geojson';
import maplibregl, { type GeoJSONSource } from 'maplibre-gl';
import { firstValueFrom } from 'rxjs';

import { RouteApi, type RouteProfile, type RouteSummary } from './route-api';

type CoordinateState = {
  fromLat: number | null;
  fromLng: number | null;
  toLat: number | null;
  toLng: number | null;
};

const DEFAULT_COORDINATES = {
  fromLat: 47.3769,
  fromLng: 8.5417,
  toLat: 47.3717,
  toLng: 8.5423,
} as const;

const SWISSTOPO_STYLE_URL =
  'https://vectortiles.geo.admin.ch/styles/ch.swisstopo.basemap.vt/style.json';

type CoordinateKey = keyof typeof DEFAULT_COORDINATES;
type MapSelectionTarget = 'start' | 'end';

const ROUTE_PROFILES = [
  { key: 'foot', label: 'Foot' },
  { key: 'car', label: 'Car' },
] as const satisfies readonly { key: RouteProfile; label: string }[];

const COORDINATE_ROWS = [
  [
    { key: 'fromLat', label: 'Start latitude' },
    { key: 'fromLng', label: 'Start longitude' },
  ],
  [
    { key: 'toLat', label: 'End latitude' },
    { key: 'toLng', label: 'End longitude' },
  ],
] as const satisfies readonly (readonly { key: CoordinateKey; label: string }[])[];

@Component({
  selector: 'app-root',
  imports: [FormsModule, DecimalPipe],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {
  private readonly routeApi = inject(RouteApi);

  protected readonly coordinateRows = COORDINATE_ROWS;
  protected readonly routeProfiles = ROUTE_PROFILES;
  protected readonly coordinates: CoordinateState = { ...DEFAULT_COORDINATES };

  protected readonly loading = signal(false);
  protected readonly errorMessage = signal<string | null>(null);
  protected readonly route = signal<RouteSummary | null>(null);
  protected readonly mapSelectionTarget = signal<MapSelectionTarget>('start');
  protected readonly routeProfile = signal<RouteProfile>('foot');

  private map: maplibregl.Map | null = null;

  constructor() {
    afterNextRender(() => {
      this.initializeMap();
    });
  }

  protected readonly canLoadRoute = (): boolean => this.hasCompleteCoordinates();

  protected async loadRoute(): Promise<void> {
    if (!this.hasCompleteCoordinates()) {
      this.errorMessage.set('Set both start and end points before loading a route.');
      return;
    }

    this.loading.set(true);
    this.errorMessage.set(null);

    const { fromLat, fromLng, toLat, toLng } = this.coordinates;

    try {
      const route = await firstValueFrom(
        this.routeApi.getRoute(
          this.routeProfile(),
          fromLat,
          fromLng,
          toLat,
          toLng,
        ),
      );

      this.route.set(route);
      this.drawRoute(route.geometry);
    } catch (error) {
      const message =
        error instanceof HttpErrorResponse
          ? error.status === 200
            ? 'Received HTTP 200 with a non-JSON response. Check the Angular proxy and GraphHopper response body.'
            : `Request failed with status ${error.status || 'unknown'}`
          : 'Route lookup failed';

      this.errorMessage.set(message);
    } finally {
      this.loading.set(false);
    }
  }

  protected setMapSelectionTarget(target: MapSelectionTarget): void {
    this.mapSelectionTarget.set(target);
  }

  protected setRouteProfile(profile: RouteProfile): void {
    if (this.loading() || this.routeProfile() === profile) {
      return;
    }

    this.routeProfile.set(profile);
    this.resetRouteState();

    if (this.hasCompleteCoordinates()) {
      void this.loadRoute();
    }
  }

  protected async swapEndpoints(): Promise<void> {
    if (!this.hasCompleteCoordinates() || this.loading()) {
      return;
    }

    const nextCoordinates: CoordinateState = {
      fromLat: this.coordinates.toLat,
      fromLng: this.coordinates.toLng,
      toLat: this.coordinates.fromLat,
      toLng: this.coordinates.fromLng,
    };

    Object.assign(this.coordinates, nextCoordinates);
    this.resetRouteState();
    this.updateWaypointSource();
    await this.loadRoute();
  }

  protected clearMapSelection(): void {
    if (this.loading()) {
      return;
    }

    Object.assign(this.coordinates, {
      fromLat: null,
      fromLng: null,
      toLat: null,
      toLng: null,
    } satisfies CoordinateState);

    this.mapSelectionTarget.set('start');
    this.resetRouteState();
    this.updateWaypointSource();
  }

  private initializeMap(): void {
    this.map = new maplibregl.Map({
      container: 'map',
      style: SWISSTOPO_STYLE_URL,
      center: [DEFAULT_COORDINATES.fromLng, DEFAULT_COORDINATES.fromLat],
      zoom: 13,
    });

    this.map.addControl(new maplibregl.NavigationControl(), 'top-right');
    this.map.addControl(
      new maplibregl.ScaleControl({
        maxWidth: 120,
        unit: 'metric',
      }),
      'bottom-left',
    );

    this.map.on('click', (event) => {
      this.updateCoordinatesFromMap(event.lngLat.lat, event.lngLat.lng);
    });

    this.map.on('load', () => {
      if (!this.map || this.map.getSource('route') || this.map.getSource('waypoints')) {
        return;
      }

      this.map.addSource('route', {
        type: 'geojson',
        data: this.createEmptyFeatureCollection(),
      });

      this.map.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        layout: {
          'line-cap': 'round',
          'line-join': 'round',
        },
        paint: {
          'line-color': '#0f766e',
          'line-width': 6,
          'line-opacity': 0.9,
        },
      });

      this.map.addSource('waypoints', {
        type: 'geojson',
        data: this.createWaypointFeatureCollection(),
      });

      this.map.addLayer({
        id: 'waypoint-points',
        type: 'circle',
        source: 'waypoints',
        paint: {
          'circle-color': ['match', ['get', 'role'], 'start', '#0f766e', '#b45309'],
          'circle-radius': 7,
          'circle-stroke-color': '#ffffff',
          'circle-stroke-width': 2,
        },
      });

      this.map.addLayer({
        id: 'waypoint-labels',
        type: 'symbol',
        source: 'waypoints',
        layout: {
          'text-field': ['get', 'label'],
          'text-size': 12,
          'text-offset': [0, 1.4],
          'text-anchor': 'top',
        },
        paint: {
          'text-color': '#0f172a',
          'text-halo-color': '#ffffff',
          'text-halo-width': 1.2,
        },
      });

      this.updateWaypointSource();
    });
  }

  private drawRoute(geometry: LineString): void {
    if (!this.map) {
      return;
    }

    const coordinates = geometry.coordinates as [number, number][];
    const featureCollection: FeatureCollection<LineString> = {
      type: 'FeatureCollection',
      features: [
        {
          type: 'Feature',
          geometry,
          properties: {},
        },
      ],
    };

    const updateSource = () => {
      if (!this.map || coordinates.length === 0) {
        return;
      }

      const source = this.map.getSource('route') as GeoJSONSource | undefined;
      if (!source) {
        return;
      }

      source.setData(featureCollection);

      const bounds = coordinates.reduce(
        (currentBounds: maplibregl.LngLatBounds, coordinate: [number, number]) => {
          return currentBounds.extend(coordinate);
        },
        new maplibregl.LngLatBounds(coordinates[0], coordinates[0]),
      );

      this.map.fitBounds(bounds, {
        padding: 48,
        duration: 800,
      });
    };

    if (this.map.isStyleLoaded()) {
      updateSource();
      return;
    }

    this.map.once('load', updateSource);
  }

  protected updateCoordinate(key: CoordinateKey, value: number | string): void {
    const nextValue = typeof value === 'number' ? value : Number(value);

    this.coordinates[key] = Number.isFinite(nextValue) ? nextValue : null;
    this.resetRouteState();
    this.updateWaypointSource();
  }

  private updateCoordinatesFromMap(lat: number, lng: number): void {
    if (this.loading()) {
      return;
    }

    const target = this.mapSelectionTarget();

    if (target === 'start') {
      this.coordinates.fromLat = lat;
      this.coordinates.fromLng = lng;
      this.mapSelectionTarget.set('end');
    } else {
      this.coordinates.toLat = lat;
      this.coordinates.toLng = lng;
      this.mapSelectionTarget.set('start');
    }

    this.resetRouteState();
    this.updateWaypointSource();

    if (this.hasCompleteCoordinates()) {
      void this.loadRoute();
    }
  }

  private resetRouteState(): void {
    this.route.set(null);
    this.errorMessage.set(null);
    this.clearRouteSource();
  }

  private clearRouteSource(): void {
    if (!this.map) {
      return;
    }

    const source = this.map.getSource('route') as GeoJSONSource | undefined;
    source?.setData(this.createEmptyFeatureCollection());
  }

  private updateWaypointSource(): void {
    if (!this.map) {
      return;
    }

    const source = this.map.getSource('waypoints') as GeoJSONSource | undefined;
    source?.setData(this.createWaypointFeatureCollection());
  }

  private createWaypointFeatureCollection(): FeatureCollection<Point> {
    const features = [] as FeatureCollection<Point>['features'];

    if (this.coordinates.fromLat !== null && this.coordinates.fromLng !== null) {
      features.push({
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: [this.coordinates.fromLng, this.coordinates.fromLat],
        },
        properties: {
          role: 'start',
          label: 'Start',
        },
      });
    }

    if (this.coordinates.toLat !== null && this.coordinates.toLng !== null) {
      features.push({
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: [this.coordinates.toLng, this.coordinates.toLat],
        },
        properties: {
          role: 'end',
          label: 'End',
        },
      });
    }

    return {
      type: 'FeatureCollection',
      features,
    };
  }

  private createEmptyFeatureCollection(): FeatureCollection<LineString> {
    return {
      type: 'FeatureCollection',
      features: [],
    };
  }

  private hasCompleteCoordinates(): this is this & {
    coordinates: {
      fromLat: number;
      fromLng: number;
      toLat: number;
      toLng: number;
    };
  } {
    return (
      this.coordinates.fromLat !== null &&
      this.coordinates.fromLng !== null &&
      this.coordinates.toLat !== null &&
      this.coordinates.toLng !== null
    );
  }
}
