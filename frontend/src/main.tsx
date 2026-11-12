import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import { restoreSession } from './api/client';
// Self-hosted brand fonts (latin subset only): no third-party request, no
// render-blocking stylesheet, and a strict font-src 'self' CSP.
import '@fontsource/jost/latin-400.css';
import '@fontsource/jost/latin-500.css';
import '@fontsource/jost/latin-600.css';
import '@fontsource/cormorant-garamond/latin-500.css';
import '@fontsource/cormorant-garamond/latin-600.css';
import '@fontsource/cinzel/latin-500.css';
import '@fontsource/cinzel/latin-600.css';
import './styles/fonts.css';
import './styles/global.css';
import './styles/layout.css';
import './styles/overlay.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60_000,
      gcTime: 10 * 60_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

void restoreSession();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
