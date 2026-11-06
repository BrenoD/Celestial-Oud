import { lazy } from 'react';
import { createBrowserRouter, Navigate, RouterProvider } from 'react-router-dom';
import { StoreLayout, CheckoutLayout } from '@/components/layout/Layout';
import { ErrorBoundary } from '@/components/ui/ErrorBoundary';

// Route-level code splitting: each page is its own chunk, so the home page
// does not pay for checkout or account code.
const Home = lazy(() => import('@/pages/Home'));
const Collection = lazy(() => import('@/pages/Collection'));
const Product = lazy(() => import('@/pages/Product'));
const Basket = lazy(() => import('@/pages/Basket'));
const Checkout = lazy(() => import('@/pages/Checkout'));
const Confirmation = lazy(() => import('@/pages/Confirmation'));
const Account = lazy(() => import('@/pages/Account'));
const Help = lazy(() => import('@/pages/Help'));
const About = lazy(() => import('@/pages/About'));
const Wishlist = lazy(() => import('@/pages/Wishlist'));
const Legal = lazy(() => import('@/pages/Legal'));
const Quiz = lazy(() => import('@/pages/Quiz'));
const NotFound = lazy(() => import('@/pages/NotFound'));

const router = createBrowserRouter([
  {
    element: <StoreLayout />,
    errorElement: <ErrorBoundary><NotFound /></ErrorBoundary>,
    children: [
      { path: '/', element: <Home /> },
      { path: '/collections', element: <Navigate to="/collections/all" replace /> },
      { path: '/collections/:slug', element: <Collection /> },
      { path: '/search', element: <Collection search /> },
      { path: '/products/:slug', element: <Product /> },
      { path: '/basket', element: <Basket /> },
      { path: '/wishlist', element: <Wishlist /> },
      { path: '/account', element: <Account /> },
      { path: '/account/:tab', element: <Account /> },
      { path: '/help', element: <Help /> },
      { path: '/about', element: <About /> },
      { path: '/quiz', element: <Quiz /> },
      { path: '/legal/:page', element: <Legal /> },
      { path: '*', element: <NotFound /> },
    ],
  },
  {
    element: <CheckoutLayout />,
    children: [
      { path: '/checkout', element: <Checkout /> },
      { path: '/checkout/confirmed/:id', element: <Confirmation /> },
    ],
  },
]);

export default function App() {
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
    </ErrorBoundary>
  );
}
