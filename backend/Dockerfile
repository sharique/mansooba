# ── builder ────────────────────────────────────────────────────────────────────
# golang:alpine tracks the latest stable Go release.
# We need CGO for mattn/go-sqlite3 (wraps the C SQLite library), so we install
# gcc and musl-dev from the Alpine package index.
FROM golang:alpine AS builder

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
# TLS outbound calls (e.g. future webhook integrations).
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
