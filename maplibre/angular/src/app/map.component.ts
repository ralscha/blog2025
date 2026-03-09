import {
  Component,
  ChangeDetectionStrategy,
  computed,
  signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  ControlComponent,
  MapComponent as NgxMapComponent,
  MarkerComponent,
  NavigationControlDirective,
  PopupComponent,
  ScaleControlDirective,
} from '@maplibre/ngx-maplibre-gl';
import { LngLatBounds, type LngLatBoundsLike } from 'maplibre-gl';

interface Mountain {
  name: string;
  coordinates: [number, number];
  elevation: number;
  range: string;
  firstAscent: string;
  description: string;
}

@Component({
  selector: 'app-map',
  imports: [
    FormsModule,
    NgxMapComponent,
    ControlComponent,
    NavigationControlDirective,
    ScaleControlDirective,
    MarkerComponent,
    PopupComponent,
  ],
  templateUrl: './map.component.html',
  styleUrl: './map.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MapComponent {
  protected readonly mapStyle =
    'https://vectortiles.geo.admin.ch/styles/ch.swisstopo.basemap.vt/style.json';
  protected readonly initialCenter: [number, number] = [8.2312, 46.8182];
  protected readonly initialZoom: [number] = [7];
  protected readonly fitBoundsOptions = {
    padding: 50,
    maxZoom: 10,
    duration: 1000,
  };

  protected readonly showMarkers = signal(false);

  private readonly mountains: Mountain[] = [
    {
      name: 'Dufourspitze (Monte Rosa)',
      coordinates: [7.8669, 45.9367],
      elevation: 4634,
      range: 'Pennine Alps',
      firstAscent: '1855',
      description:
        'The highest peak in Switzerland, part of the Monte Rosa massif on the border with Italy.',
    },
    {
      name: 'Dom',
      coordinates: [7.8588, 46.0938],
      elevation: 4545,
      range: 'Pennine Alps',
      firstAscent: '1858',
      description:
        'The highest mountain lying entirely within Switzerland, located in the Mischabel group.',
    },
    {
      name: 'Liskamm',
      coordinates: [7.8356, 45.9225],
      elevation: 4527,
      range: 'Pennine Alps',
      firstAscent: '1861',
      description:
        "A distinctive mountain with a sharp east-west oriented ridge, nicknamed 'the Man-eater'.",
    },
    {
      name: 'Weisshorn',
      coordinates: [7.7165, 46.1058],
      elevation: 4506,
      range: 'Pennine Alps',
      firstAscent: '1861',
      description:
        'One of the most beautiful peaks in the Alps, with a distinctive pyramidal shape.',
    },
    {
      name: 'Matterhorn',
      coordinates: [7.6586, 45.9764],
      elevation: 4478,
      range: 'Pennine Alps',
      firstAscent: '1865',
      description:
        'The most famous mountain in Switzerland, known for its distinctive pyramid shape.',
    },
  ];

  protected readonly visibleMountains = computed(() =>
    this.showMarkers() ? this.mountains : [],
  );

  protected readonly center = computed<[number, number]>(() =>
    this.showMarkers() ? [7.7873, 46.007] : this.initialCenter,
  );

  protected readonly zoom = computed<[number]>(() =>
    this.showMarkers() ? [8] : this.initialZoom,
  );

  protected readonly bounds = computed<LngLatBoundsLike | null>(() => {
    if (!this.showMarkers()) {
      return null;
    }

    const bounds = new LngLatBounds();
    this.mountains.forEach((mountain) => {
      bounds.extend(mountain.coordinates);
    });

    return bounds;
  });

  protected toggleMarkers(event: Event): void {
    const target = event.target as HTMLInputElement;
    this.showMarkers.set(target.checked);
  }
}
