import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './guards/auth.guard';

export const routes: Routes = [
  { path: '', redirectTo: '/todos', pathMatch: 'full' },
  {
    path: 'sign-in',
    loadComponent: () => import('./auth/sign-in').then((m) => m.SignInComponent),
    canActivate: [guestGuard],
  },
  {
    path: 'sign-up',
    loadComponent: () => import('./auth/sign-up').then((m) => m.SignUpComponent),
    canActivate: [guestGuard],
  },
  {
    path: 'profile',
    loadComponent: () => import('./profile/profile').then((m) => m.ProfileComponent),
    canActivate: [authGuard],
  },
  {
    path: 'todos',
    loadComponent: () => import('./todos/todo-list').then((m) => m.TodoListComponent),
    canActivate: [authGuard],
  },
  {
    path: 'todos/:id',
    loadComponent: () => import('./todos/todo-edit').then((m) => m.TodoEditComponent),
    canActivate: [authGuard],
  },
  { path: '**', redirectTo: '/todos' },
];
