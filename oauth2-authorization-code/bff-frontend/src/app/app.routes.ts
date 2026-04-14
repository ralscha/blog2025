import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./bff-home.page').then((module) => module.BffBrowserSessionPage),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
