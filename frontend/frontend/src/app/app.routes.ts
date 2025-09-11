import { Routes } from '@angular/router';
import { Auth } from './auth/auth';
import { AuthLayout } from './layouts/auth-layout/auth-layout';

export const routes: Routes = [
  {
    path: '',
    component: AuthLayout,
    children: [{ path: 'auth', component: Auth }],
  },
  {
    path: '',
    loadChildren: () =>
      import('./layouts/main-layout/main-layout.routes').then(
        (mod) => mod.PAGES_ROUTES
      ),
    // canActivate: [() => inject(Authservice).isLoggedIn],
  },
  {
    path: '**',
    redirectTo: 'home',
  },
];
