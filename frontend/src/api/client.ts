/**
 * Thin fetch wrapper for the Go API.
 *
 * - JSON in / JSON out, typed.
 * - Bearer token from the auth store when present.
 * - Timeout via AbortController so a slow backend never hangs the UI.
 * - Retries idempotent GETs with exponential backoff on 5xx / network errors.
 *   Writes are never retried here; callers that need safe retries send an
 *   Idempotency-Key (see orders.create).
 */
import { config } from '@/config';
import { useAuthStore } from '@/store/auth';

export class ApiError extends Error {
  status: number;
  code?: string;
  details?: unknown;
  constructor(status: number, message: string, code?: string, details?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  headers?: Record<string, string>;
  timeoutMs?: number;
  retries?: number;
  signal?: AbortSignal;
  /** Internal: set on the single retry after a token refresh (never sent to the server). */
  _authRetried?: boolean;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const method = opts.method ?? 'GET';
  const retries = opts.retries ?? (method === 'GET' ? 2 : 0);
  const timeoutMs = opts.timeoutMs ?? 10_000;

  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...opts.headers,
  };
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
  const token = useAuthStore.getState().token;
  if (token) headers.Authorization = `Bearer ${token}`;

  let attempt = 0;
  for (;;) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    const onOuterAbort = () => controller.abort();
    opts.signal?.addEventListener('abort', onOuterAbort, { once: true });

    try {
      const res = await fetch(`${config.apiUrl}${path}`, {
        method,
        headers,
        body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
        signal: controller.signal,
        credentials: 'omit',
      });

      if (res.status === 401 && !opts._authRetried) {
        const refreshed = await tryRefresh();
        if (refreshed) {
          return request<T>(path, { ...opts, _authRetried: true });
        }
        useAuthStore.getState().signOut();
      }

      if (!res.ok) {
        let payload: { message?: string; code?: string; details?: unknown } = {};
        try {
          payload = await res.json();
        } catch {
          /* non-JSON error body */
        }
        const err = new ApiError(res.status, payload.message ?? res.statusText, payload.code, payload.details);
        if (res.status >= 500 && attempt < retries) throw err; // retry below
        throw Object.assign(err, { final: true });
      }

      if (res.status === 204) return undefined as T;
      return (await res.json()) as T;
    } catch (e) {
      const final = (e as { final?: boolean }).final;
      const aborted = opts.signal?.aborted;
      if (final || aborted || attempt >= retries) throw e;
      attempt += 1;
      await sleep(250 * 2 ** attempt + Math.random() * 100);
    } finally {
      clearTimeout(timer);
      opts.signal?.removeEventListener('abort', onOuterAbort);
    }
  }
}

// Exchange the refresh token once; returns false when the session is gone.
let refreshing: Promise<boolean> | null = null;
async function tryRefresh(): Promise<boolean> {
  const { refreshToken, setTokens } = useAuthStore.getState();
  if (!refreshToken || !config.apiUrl) return false;
  refreshing ??= fetch(`${config.apiUrl}/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken }),
  })
    .then(async (r) => {
      if (!r.ok) return false;
      const data = (await r.json()) as { token: string; refreshToken?: string };
      setTokens(data.token, data.refreshToken ?? null);
      return true;
    })
    .catch(() => false)
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

export function qs(params: Record<string, unknown>): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue;
    if (Array.isArray(v)) {
      if (v.length) sp.set(k, v.join(','));
    } else sp.set(k, String(v));
  }
  const s = sp.toString();
  return s ? `?${s}` : '';
}
