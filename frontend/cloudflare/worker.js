// Edge entry for the storefront: canonical host, then static assets (SPA).
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    if (url.hostname === 'www.celestialoud.com') {
      url.hostname = 'celestialoud.com';
      return Response.redirect(url.toString(), 301);
    }
    return env.ASSETS.fetch(request);
  },
};
