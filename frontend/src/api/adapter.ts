/**
 * Build-time adapter switch. vite.config.ts aliases this module to ./mock.ts
 * when VITE_API_URL is empty, so production bundles never contain the mock
 * catalogue (and the preview never loads the real client). TypeScript sees
 * the real adapter.
 */
export { api } from './real';
