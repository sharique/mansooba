# ── builder ────────────────────────────────────────────────────────────────────
# Pinned to the Go release line (not an exact patch — a rebuild picks up patch
# and security releases; no floating tags).
# The line must match the go directive in go.mod and the Go version in CI;
# scripts/verify-versions.sh enforces that. We need CGO for mattn/go-sqlite3
# (wraps the C SQLite library), so we install gcc and musl-dev from the
# Alpine package index.
FROM golang:1.27-alpine3.24 AS builder

ENV CGO_ENABLED=1

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Download dependencies in a separate layer so Docker cache is reused on
# source-only changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# APP_VERSION is baked into the binary, so
# a user can find the matching source for what they're running via
# GET /api/v1/about. Defaults to "dev" for a local, non-CI build.
ARG APP_VERSION=dev
RUN go build -ldflags="-s -w -X main.version=${APP_VERSION}" -o /app/server ./cmd/server

# ── final ──────────────────────────────────────────────────────────────────────
# Alpine keeps the image small (~10 MB). ca-certificates is needed for any
# TLS outbound calls (e.g. future webhook integrations). tzdata is needed
# because Alpine ships no IANA timezone database at all — without it,
# Go's time.LoadLocation() fails for every real zone name (only "UTC"
# works), which is exactly what broke profile timezone updates.
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

LABEL org.opencontainers.image.licenses="AGPL-3.0-only"
LABEL org.opencontainers.image.source="https://github.com/sharique/mansooba"

WORKDIR /app
COPY --from=builder /app/server .
# Populated at CI build time by scripts/generate-notices.sh;
# the placeholder keeps `docker compose build` working locally.
COPY legal/ /licenses/

EXPOSE 8080
CMD ["./server"]
