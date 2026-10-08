# ── builder ────────────────────────────────────────────────────────────────────
FROM node:24-alpine3.24 AS builder

WORKDIR /app

# Install dependencies in a separate layer for cache efficiency.
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# NUXT_PUBLIC_API_BASE_URL is set to a relative path so that the browser sends
# API requests to the same origin; nginx then proxies /api/ to the backend.
# Override at build time with: docker build --build-arg NUXT_PUBLIC_API_BASE_URL=...
ARG NUXT_PUBLIC_API_BASE_URL=/api/v1
ENV NUXT_PUBLIC_API_BASE_URL=${NUXT_PUBLIC_API_BASE_URL}

RUN npm run generate

# ── final ──────────────────────────────────────────────────────────────────────
# nginx:alpine serves the static Nuxt output and proxies /api/ to the backend.
FROM nginx:1.30-alpine3.24

LABEL org.opencontainers.image.licenses="AGPL-3.0-only"
LABEL org.opencontainers.image.source="https://github.com/sharique/mansooba"

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/.output/public /usr/share/nginx/html
# Populated at CI build time by scripts/generate-notices.sh;
# the placeholder keeps `docker compose build` working locally.
# Deliberately outside the nginx web root (/usr/share/nginx/html) — this
# matches the backend image's /licenses/ path, not something nginx serves.
COPY legal/ /licenses/

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
