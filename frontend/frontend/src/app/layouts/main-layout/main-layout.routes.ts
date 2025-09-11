import { Route } from '@angular/router';
import { MainLayout } from './main-layout';

export const PAGES_ROUTES: Route[] = [
  {
    path: '',
    redirectTo: 'properties',
    pathMatch: 'full',
  },
  {
    path: '',
    component: MainLayout,
    children: [
      {
        path: 'properties',
        loadComponent: () =>
          import('../../pages/properties/properties').then((m) => m.Properties),
      },
    ],
  },
];
