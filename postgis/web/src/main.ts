import './style.css';
import 'maplibre-gl/dist/maplibre-gl.css';
import { DemoApp } from './ui';

const root = document.querySelector<HTMLDivElement>('#app');

if (!root) {
  throw new Error('App root not found.');
}

new DemoApp(root).mount();
