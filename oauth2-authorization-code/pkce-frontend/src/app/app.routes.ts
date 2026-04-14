import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./pkce-home.page').then((module) => module.PkceBrowserFlowPage),
  },
  {
    path: 'callback',
    loadComponent: () =>
      import('./pkce-callback.page').then((module) => module.PkceBrowserCallbackPage),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
