/**
 * Worker in front of the Go API container.
 *
 * - Routes every request to a container instance (round-robin over N
 *   instances) so traffic spreads horizontally.
 * - Caches public GETs at the edge (collections, products, content) using the
 *   Cache-Control the Go API sets — the origin is hit once per POP per TTL.
 * - Fires the cron tick into the container every 5 minutes.
 */
import { Container, getRandom } from "@cloudflare/containers";

export class ApiContainer extends Container<Env> {
  defaultPort = 8080;
  sleepAfter = "10m";
  envVars = {
    APP_ENV: this.env.APP_ENV,
    PUBLIC_URL: this.env.PUBLIC_URL,
    API_URL: this.env.API_URL,
    CORS_ORIGINS: this.env.CORS_ORIGINS,
    EMAIL_FROM: this.env.EMAIL_FROM,
    DATABASE_URL: this.env.HYPERDRIVE?.connectionString ?? this.env.DATABASE_URL,
    JWT_SECRET: this.env.JWT_SECRET,
    STRIPE_SECRET_KEY: this.env.STRIPE_SECRET_KEY ?? "",
    STRIPE_PUBLISHABLE_KEY: this.env.STRIPE_PUBLISHABLE_KEY ?? "",
    STRIPE_WEBHOOK_SECRET: this.env.STRIPE_WEBHOOK_SECRET ?? "",
    RESEND_API_KEY: this.env.RESEND_API_KEY ?? "",
    ADMIN_EMAILS: this.env.ADMIN_EMAILS ?? "",
    PAYMENTS_MODE: this.env.PAYMENTS_MODE ?? "",
    CRON_SECRET: this.env.CRON_SECRET,
    IDEAL_POSTCODES_KEY: this.env.IDEAL_POSTCODES_KEY ?? "",
  };
}

interface Env {
  API: DurableObjectNamespace<ApiContainer>;
  HYPERDRIVE?: { connectionString: string };
  APP_ENV: string;
  PUBLIC_URL: string;
  API_URL: string;
  CORS_ORIGINS: string;
  EMAIL_FROM: string;
  DATABASE_URL: string;
  JWT_SECRET: string;
  STRIPE_SECRET_KEY?: string;
  STRIPE_PUBLISHABLE_KEY?: string;
  STRIPE_WEBHOOK_SECRET?: string;
  RESEND_API_KEY?: string;
  ADMIN_EMAILS?: string;
  PAYMENTS_MODE?: string;
  CRON_SECRET: string;
  IDEAL_POSTCODES_KEY?: string;
}

const INSTANCES = 3; // warm instances to spread load across; max_instances caps bursts
const CACHEABLE = /^\/v1\/(collections|products|notes|delivery-methods|content|blog|settings)(\/|$)/;

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);

    // Edge cache for anonymous public reads.
    const cacheable = request.method === "GET" && CACHEABLE.test(url.pathname) && !request.headers.has("Authorization");
    const cache = caches.default;
    if (cacheable) {
      const hit = await cache.match(request);
      if (hit) return hit;
    }

    const container = await getRandom(env.API, INSTANCES);
    const response = await container.fetch(request);

    if (cacheable && response.ok && response.headers.get("Cache-Control")?.includes("max-age")) {
      ctx.waitUntil(cache.put(request, response.clone()));
    }
    return response;
  },

  async scheduled(_event: ScheduledEvent, env: Env, ctx: ExecutionContext) {
    const container = await getRandom(env.API, 1);
    ctx.waitUntil(
      container.fetch(new Request(`${env.API_URL}/internal/cron/tick`, { method: "POST", headers: { "X-Cron-Secret": env.CRON_SECRET } })),
    );
  },
};
