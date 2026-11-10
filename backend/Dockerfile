# ---- build ----
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- run ----
# Distroless static: ~10 MB image, no shell, runs as non-root. Cloudflare
# Containers (and any OCI runtime) start it directly.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
ENV PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/api"]
