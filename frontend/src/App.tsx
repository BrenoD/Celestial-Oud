import { lazy, Suspense, type ReactNode } from "react";
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
} from "react-router-dom";
import {
  StoreLayout,
  CheckoutLayout,
  RootLayout,
} from "@/components/layout/Layout";
import { ErrorBoundary } from "@/components/ui/ErrorBoundary";

// Storefront pages are in the entry bundle (~17 kB gzipped in total): loading
// them lazily added a network round trip before the main heading could paint
// (LCP) on every first visit. The admin back office is far bigger and only
// for staff, so it stays code-split.
import Home from "@/pages/Home";
import Collection from "@/pages/Collection";
import Product from "@/pages/Product";
import Basket from "@/pages/Basket";
import Checkout from "@/pages/Checkout";
import Confirmation from "@/pages/Confirmation";
import Account from "@/pages/Account";
import Help from "@/pages/Help";
import About from "@/pages/About";
import Wishlist from "@/pages/Wishlist";
import Legal from "@/pages/Legal";
import Quiz from "@/pages/Quiz";
import NotFound from "@/pages/NotFound";
import Newsletter from "@/pages/Newsletter";
const AdminLayout = lazy(() => import("@/admin/AdminLayout"));
const AdminSection = lazy(() => import("@/admin/sections"));

/** Boundary for lazily loaded top-level layouts (store/checkout layouts are eager and own their Suspense). */
function Lazy({ children }: { children: ReactNode }) {
  return (
    <Suspense
      fallback={
        <main className="container page" aria-busy="true">
          <div
            className="skeleton"
            style={{ height: 44, width: 320, marginBottom: 24 }}
          />
          <div className="skeleton" style={{ height: 420 }} />
        </main>
      }
    >
      {children}
    </Suspense>
  );
}

const router = createBrowserRouter(
  [
    {
      element: <RootLayout />,
      children: [
        {
          element: <StoreLayout />,
          errorElement: (
            <ErrorBoundary>
              <NotFound />
            </ErrorBoundary>
          ),
          children: [
            { path: "/", element: <Home /> },
            {
              path: "/collections",
              element: <Navigate to="/collections/all" replace />,
            },
            { path: "/collections/:slug", element: <Collection /> },
            { path: "/search", element: <Collection search /> },
            { path: "/products/:slug", element: <Product /> },
            { path: "/basket", element: <Basket /> },
            { path: "/wishlist", element: <Wishlist /> },
            { path: "/account", element: <Account /> },
            { path: "/account/:tab", element: <Account /> },
            { path: "/help", element: <Help /> },
            { path: "/about", element: <About /> },
            { path: "/quiz", element: <Quiz /> },
            { path: "/legal/:page", element: <Legal /> },
            { path: "/newsletter/:action", element: <Newsletter /> },
            { path: "*", element: <NotFound /> },
          ],
        },
        {
          path: "/admin",
          element: (
            <Lazy>
              <AdminLayout />
            </Lazy>
          ),
          errorElement: (
            <Lazy>
              <NotFound />
            </Lazy>
          ),
          children: [
            {
              index: true,
              element: <Navigate to="/admin/dashboard" replace />,
            },
            { path: ":section", element: <AdminSection /> },
          ],
        },
        {
          element: <CheckoutLayout />,
          children: [
            { path: "/checkout", element: <Checkout /> },
            { path: "/checkout/confirmed/:id", element: <Confirmation /> },
          ],
        },
      ],
    },
  ],
  {
    // Honour Vite's base path so the app works from a sub-folder (GitHub Pages).
    basename: import.meta.env.BASE_URL.replace(/\/$/, "") || "/",
  },
);

export default function App() {
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
    </ErrorBoundary>
  );
}
