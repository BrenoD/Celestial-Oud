/// <reference types="vitest/config" />
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import type { Plugin } from 'vite';

/** Preloads the fonts used above the fold (body text and headings). */
function preloadFonts(patterns: RegExp[]): Plugin {
  return {
    name: 'preload-fonts',
    apply: 'build',
    transformIndexHtml: {
      order: 'post',
      handler(_html, ctx) {
        const files = Object.keys(ctx.bundle ?? {}).filter((f) => f.endsWith('.woff2') && patterns.some((p) => p.test(f)));
        return files.map((f) => ({
          tag: 'link',
          attrs: { rel: 'preload', href: `${ctx.server ? '' : (process.env.VITE_BASE_PATH || '/')}${f}`, as: 'font', type: 'font/woff2', crossorigin: '' },
          injectTo: 'head' as const,
        }));
      },
    },
  };
}

// Static build. Every asset under /assets gets a content hash so nginx/CDN can
// cache it forever; index.html is the only file that must not be cached.
export default defineConfig(({ mode }) => {
  const env = { ...loadEnv(mode, process.cwd(), 'VITE_'), ...process.env };
  const mockApi = !env.VITE_API_URL;
  return {
  // Sub-path deploys (e.g. GitHub Pages at /Celestial-Oud/) set VITE_BASE_PATH.
  base: process.env.VITE_BASE_PATH || '/',
  plugins: [react(), preloadFonts([/jost-latin-400-normal/, /cormorant-garamond-latin-600-normal/])],
  resolve: {
    alias: [
      // Pick the API adapter at build time (see src/api/adapter.ts).
      { find: /^(@\/api|\.)\/adapter$/, replacement: mockApi ? '/src/api/mock.ts' : '/src/api/real.ts' },
      { find: '@', replacement: '/src' },
    ],
  },
  build: {
    target: 'es2020',
    sourcemap: false,
    cssCodeSplit: true,
    rollupOptions: {
      output: {
        // Long-lived vendor chunks: a release that only touches app code does
        // not invalidate React/router/query in visitors' caches.
        manualChunks: {
          react: ['react', 'react-dom', 'react-router-dom'],
          query: ['@tanstack/react-query'],
        },
      },
    },
  },
  server: { port: 5173, strictPort: false },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
};
});
