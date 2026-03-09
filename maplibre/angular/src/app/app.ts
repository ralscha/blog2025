import { Component, signal } from '@angular/core';
import { MapComponent } from './map.component';

@Component({
  selector: 'app-root',
  imports: [MapComponent],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {
  protected readonly title = signal('Swiss Topo Map Example');
}
