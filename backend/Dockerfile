# ── builder ────────────────────────────────────────────────────────────────────
# Pinned to an exact Go release (ADR-034: no floating tags). The version must
# match the go directive in go.mod and the Go version in CI; scripts/verify-versions.sh
# enforces that. We need CGO for mattn/go-sqlite3 (wraps the C SQLite library), so we install
# gcc and musl-dev from the Alpine package index.
FROM golang:1.27-alpine3.24 AS builder

ENV CGO_ENABLED=1

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Download dependencies in a separate layer so Docker cache is reused on
# source-only changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /app/server ./cmd/server

# ── final ──────────────────────────────────────────────────────────────────────
# Alpine keeps the image small (~10 MB). ca-certificates is needed for any
# TLS outbound calls (e.g. future webhook integrations). tzdata is needed
# because Alpine ships no IANA timezone database at all — without it,
# Go's time.LoadLocation() fails for every real zone name (only "UTC"
# works), which is exactly what broke profile timezone updates.
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
