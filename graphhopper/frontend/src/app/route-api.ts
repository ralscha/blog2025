import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import type { LineString } from 'geojson';
import { map } from 'rxjs';

import { environment } from '../environments/environment';

export type RouteProfile = 'car' | 'foot';

export interface RouteSummary {
  distanceMeters: number;
  timeMillis: number;
  geometry: LineString;
}

interface GraphhopperPath {
  distance: number;
  time: number;
  points: LineString;
}

interface GraphhopperRouteResponse {
  paths: GraphhopperPath[];
}

@Injectable({ providedIn: 'root' })
export class RouteApi {
  private readonly http = inject(HttpClient);
  private readonly routeUrl = `${environment.graphhopperBaseUrl}/route`;

  getRoute(profile: RouteProfile, fromLat: number, fromLng: number, toLat: number, toLng: number) {
    const params = new HttpParams()
      .append('point', `${fromLat},${fromLng}`)
      .append('point', `${toLat},${toLng}`)
      .set('profile', profile)
      .set('instructions', false)
      .set('points_encoded', false);

    return this.http.get<GraphhopperRouteResponse>(this.routeUrl, { params }).pipe(
      map((response) => {
        const [path] = response.paths ?? [];

        if (!path) {
          throw new Error('GraphHopper returned no route');
        }

        return {
          distanceMeters: path.distance,
          timeMillis: path.time,
          geometry: path.points,
        } satisfies RouteSummary;
      }),
    );
  }
}
