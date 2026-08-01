package lokiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/pkg/lokiclient"
)

func TestClient_Push_SendsExpectedRequest(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode push body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	ts := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	err := c.Push(context.Background(), map[string]string{"app": "mansooba", "category": "authentication"}, ts, `{"action":"login_failed"}`)
	if err != nil {
		t.Fatalf("Push returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/loki/api/v1/push" {
		t.Errorf("expected /loki/api/v1/push, got %s", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("expected application/json content type, got %s", gotContentType)
	}

	streams, ok := gotBody["streams"].([]any)
	if !ok || len(streams) != 1 {
		t.Fatalf("expected exactly one stream, got %v", gotBody["streams"])
	}
	stream := streams[0].(map[string]any)
	labels := stream["stream"].(map[string]any)
	if labels["app"] != "mansooba" || labels["category"] != "authentication" {
		t.Errorf("unexpected labels: %v", labels)
	}
	values := stream["values"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected exactly one value, got %d", len(values))
	}
	pair := values[0].([]any)
	if pair[0] != "1785585600000000000" {
		t.Errorf("expected nanosecond timestamp, got %v", pair[0])
	}
	if pair[1] != `{"action":"login_failed"}` {
		t.Errorf("unexpected line: %v", pair[1])
	}
}

func TestClient_Push_ReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	err := c.Push(context.Background(), map[string]string{"app": "mansooba"}, time.Now(), "{}")
	if err == nil {
		t.Fatal("expected an error for a 500 response, got nil")
	}
}

func TestClient_QueryRange_ParsesResponse(t *testing.T) {
	var gotQuery, gotDirection string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("query")
		gotDirection = r.URL.Query().Get("direction")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "streams",
				"result": [
					{
						"stream": {"app": "mansooba", "category": "authentication"},
						"values": [["1785585600000000000", "{\"action\":\"login_failed\"}"]]
					}
				]
			}
		}`))
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	entries, err := c.QueryRange(context.Background(), lokiclient.QueryRangeParams{
		Query: `{app="mansooba"}`,
		Start: time.Now().Add(-time.Hour),
		End:   time.Now(),
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("QueryRange returned error: %v", err)
	}
	if gotQuery != `{app="mansooba"}` {
		t.Errorf("expected query to be passed through, got %s", gotQuery)
	}
	if gotDirection != "backward" {
		t.Errorf("expected default direction 'backward', got %s", gotDirection)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Line != `{"action":"login_failed"}` {
		t.Errorf("unexpected line: %s", entries[0].Line)
	}
	if entries[0].Labels["category"] != "authentication" {
		t.Errorf("unexpected labels: %v", entries[0].Labels)
	}
	wantTS := time.Unix(0, 1785585600000000000)
	if !entries[0].Timestamp.Equal(wantTS) {
		t.Errorf("expected timestamp %v, got %v", wantTS, entries[0].Timestamp)
	}
}

func TestClient_QueryRange_ReturnsErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	_, err := c.QueryRange(context.Background(), lokiclient.QueryRangeParams{Query: `{app="mansooba"}`})
	if err == nil {
		t.Fatal("expected an error for a 400 response, got nil")
	}
}

func TestClient_Ready_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			t.Errorf("expected /ready, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	if err := c.Ready(context.Background()); err != nil {
		t.Fatalf("expected Ready to succeed, got %v", err)
	}
}

func TestClient_Ready_ReturnsErrorWhenNotReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := lokiclient.New(srv.URL)
	if err := c.Ready(context.Background()); err == nil {
		t.Fatal("expected an error when Loki is not ready, got nil")
	}
}
