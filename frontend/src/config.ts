/** Runtime configuration, read once from Vite env. */
const env = import.meta.env;

export const config = {
  /** Go API base URL. Empty string → mock adapter. */
  apiUrl: (env.VITE_API_URL ?? '').replace(/\/$/, ''),
  /** Free UK delivery threshold, in pence. */
  freeDeliveryThreshold: Number(env.VITE_FREE_DELIVERY_THRESHOLD ?? 5000),
  assetBase: (env.VITE_ASSET_BASE ?? '').replace(/\/$/, ''),
  /** Dispatch cut-off shown across the store. */
  dispatchCutoff: '2pm',
  returnsDays: 14,
  brand: {
    name: 'Celestial Oud',
    tagline: 'Fragrances',
    instagram: 'https://instagram.com/celestial_oud',
    supportEmail: 'hello@celestialoud.co.uk',
  },
} as const;

export const useMockApi = config.apiUrl === '';

export function assetUrl(path: string): string {
  if (/^https?:\/\//.test(path)) return path;
  return `${config.assetBase}${path.startsWith('/') ? path : `/${path}`}`;
}
