// Package lokiclient is a thin, dependency-free HTTP client for Grafana
// Loki's plain HTTP+JSON API (011-system-logs, ADR-031) — push, query_range,
// and a readiness check. No third-party Loki SDK is used, matching this
// project's existing thin-wrapper convention for internal/pkg/rdsclient and
// internal/pkg/attachmentstorage (research.md Decision 5).
package lokiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client talks to a single Loki instance over HTTP.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client for the given Loki base URL (e.g.
// "http://loki:3100" in docker-compose, no trailing slash required).
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		// query_range latency scales with the days queried (observed ~2.3s at
		// 90 days, ~15s at 365 days against filesystem-backed storage in local
		// testing) — 5s was too tight for the common "no date filter" case
		// even after systemlog_service.go bounds that default to the
		// retention window instead of a hardcoded wide range.
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

// Push durably writes a single log line to Loki, labeled with labels
// (low-cardinality only — research.md Decision 1) and timestamped ts. The
// line itself is expected to already be a JSON-encoded string; lokiclient
// does not interpret its contents.
func (c *Client) Push(ctx context.Context, labels map[string]string, ts time.Time, line string) error {
	body := pushRequest{
		Streams: []pushStream{
			{
				Stream: labels,
				Values: [][2]string{{strconv.FormatInt(ts.UnixNano(), 10), line}},
			},
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("lokiclient: marshal push body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/loki/api/v1/push", bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("lokiclient: build push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("lokiclient: push request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lokiclient: push returned %d: %s", resp.StatusCode, readBodySnippet(resp.Body))
	}
	return nil
}

// QueryRangeParams configures a QueryRange call. Query is a LogQL query
// string; Direction is "backward" (newest first, the default this package
// always sends) or "forward".
type QueryRangeParams struct {
	Query     string
	Start     time.Time
	End       time.Time
	Limit     int
	Direction string
}

// LogEntry is a single result row from QueryRange.
type LogEntry struct {
	Timestamp time.Time
	Line      string
	Labels    map[string]string
}

// QueryRange executes a LogQL query over [Start, End] and returns matching
// entries, newest first (when Direction is "backward", the default).
func (c *Client) QueryRange(ctx context.Context, p QueryRangeParams) ([]LogEntry, error) {
	direction := p.Direction
	if direction == "" {
		direction = "backward"
	}
	q := url.Values{}
	q.Set("query", p.Query)
	q.Set("start", strconv.FormatInt(p.Start.UnixNano(), 10))
	q.Set("end", strconv.FormatInt(p.End.UnixNano(), 10))
	q.Set("direction", direction)
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/loki/api/v1/query_range?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("lokiclient: build query_range request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lokiclient: query_range request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lokiclient: query_range returned %d: %s", resp.StatusCode, readBodySnippet(resp.Body))
	}

	var parsed queryRangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("lokiclient: decode query_range response: %w", err)
	}

	var entries []LogEntry
	for _, stream := range parsed.Data.Result {
		for _, v := range stream.Values {
			if len(v) != 2 {
				continue
			}
			nanos, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				continue
			}
			entries = append(entries, LogEntry{
				Timestamp: time.Unix(0, nanos),
				Line:      v[1],
				Labels:    stream.Stream,
			})
		}
	}
	return entries, nil
}

// Ready reports whether Loki is reachable and ready to accept traffic, via
// its /ready endpoint. Used by the extended health check (Constitution V —
// a health-check endpoint is required for every long-running service, and
// Loki is this feature's first new one, ADR-031).
func (c *Client) Ready(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ready", nil)
	if err != nil {
		return fmt.Errorf("lokiclient: build ready request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("lokiclient: ready request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lokiclient: not ready, status %d: %s", resp.StatusCode, readBodySnippet(resp.Body))
	}
	return nil
}

type pushRequest struct {
	Streams []pushStream `json:"streams"`
}

type pushStream struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string       `json:"values"`
}

type queryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][2]string       `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func readBodySnippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 512))
	return string(b)
}
