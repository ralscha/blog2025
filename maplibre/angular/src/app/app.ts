import { Component, signal, ChangeDetectionStrategy } from '@angular/core';
import { MapComponent } from './map.component';

@Component({
  selector: 'app-root',
  imports: [MapComponent],
  templateUrl: './app.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrl: './app.css',
})
export class App {
  protected readonly title = signal('Swiss Topo Map Example');
}
