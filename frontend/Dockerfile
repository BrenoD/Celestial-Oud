# ---- build ----
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
# Build-time config (baked into the static bundle).
ARG VITE_API_URL
ARG VITE_FREE_DELIVERY_THRESHOLD=5000
ARG VITE_ASSET_BASE
ENV VITE_API_URL=$VITE_API_URL \
    VITE_FREE_DELIVERY_THRESHOLD=$VITE_FREE_DELIVERY_THRESHOLD \
    VITE_ASSET_BASE=$VITE_ASSET_BASE
RUN npm run build

# ---- serve ----
# Stateless nginx serving pre-compressed static files. Scale horizontally by
# running N replicas behind a load balancer / put a CDN in front.
FROM nginx:1.27-alpine
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/storefront.conf
COPY nginx-security.conf /etc/nginx/snippets/security-headers.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
CMD ["nginx", "-g", "daemon off;"]
