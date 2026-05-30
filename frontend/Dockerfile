# ── builder ────────────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

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
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/.output/public /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
