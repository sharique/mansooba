package repository_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
	"github.com/sharique/mansooba/internal/repository"
)

func TestLokiSystemLogRepository_Create_PushesLabeledJSONLine(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	err := repo.Create(context.Background(), domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "login_failed",
		Outcome:  "failure",
		Actor:    "jane@example.com",
		Detail:   "invalid credentials",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	streams := gotBody["streams"].([]any)
	stream := streams[0].(map[string]any)
	labels := stream["stream"].(map[string]any)
	if labels["app"] != "mansooba" || labels["category"] != "authentication" {
		t.Errorf("unexpected labels: %v", labels)
	}

	values := stream["values"].([]any)
	pair := values[0].([]any)
	var line map[string]any
	if err := json.Unmarshal([]byte(pair[1].(string)), &line); err != nil {
		t.Fatalf("failed to decode pushed line: %v", err)
	}
	if line["action"] != "login_failed" || line["outcome"] != "failure" || line["actor"] != "jane@example.com" {
		t.Errorf("unexpected line body: %v", line)
	}
}

func TestLokiSystemLogRepository_FindPaginated_Unfiltered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != `{app="mansooba"}` {
			t.Errorf("expected unfiltered selector, got %q", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "streams", "result": [
				{"stream": {"app": "mansooba", "category": "authentication"},
				 "values": [
					["1785585700000000000", "{\"action\":\"login_success\",\"outcome\":\"success\",\"actor\":\"a@x.com\"}"],
					["1785585600000000000", "{\"action\":\"login_failed\",\"outcome\":\"failure\",\"actor\":\"b@x.com\"}"]
				 ]}
			]}
		}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	result, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result.Entries))
	}
	// Newest first.
	if result.Entries[0].Action != "login_success" {
		t.Errorf("expected newest entry first, got action=%s", result.Entries[0].Action)
	}
	if result.Entries[0].Category != "authentication" {
		t.Errorf("expected category from label, got %s", result.Entries[0].Category)
	}
	if result.Entries[0].ID == "" {
		t.Error("expected a synthesized, non-empty ID")
	}
}

func TestLokiSystemLogRepository_FindPaginated_CategoryFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != `{app="mansooba",category="db_lifecycle"}` {
			t.Errorf("expected category label selector, got %q", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	_, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{
		Category: domain.SystemLogCategoryDBLifecycle,
		Page:     1, Size: 20,
	})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
}

func TestLokiSystemLogRepository_FindPaginated_ActorFilterUsesJSONFieldMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		want := `{app="mansooba"} | json | actor=~ "(?i)jane@example\\.com"`
		if query != want {
			t.Errorf("expected %q, got %q", want, query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	_, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{
		Actor: "jane@example.com",
		Page:  1, Size: 20,
	})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
}

func TestLokiSystemLogRepository_FindPaginated_KeywordSearchIsBroaderLineFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		want := `{app="mansooba"} |~ "(?i)jane"`
		if query != want {
			t.Errorf("expected %q, got %q", want, query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	_, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{
		Q:    "jane",
		Page: 1, Size: 20,
	})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
}

func TestLokiSystemLogRepository_FindPaginated_CombinesFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		want := `{app="mansooba",category="authentication"} |~ "(?i)jane" | json | actor=~ "(?i)jane@x\\.com"`
		if query != want {
			t.Errorf("expected combined query %q, got %q", want, query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	_, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{
		Category: domain.SystemLogCategoryAuthentication,
		Actor:    "jane@x.com",
		Q:        "jane",
		Page:     1, Size: 20,
	})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
}

func TestLokiSystemLogRepository_FindPaginated_Pagination(t *testing.T) {
	// 5 entries, size=2: page 1 -> [0,1], page 2 -> [2,3], page 3 -> [4].
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "streams", "result": [
				{"stream": {"app": "mansooba"}, "values": [
					["1785585705000000000", "{\"action\":\"e5\"}"],
					["1785585704000000000", "{\"action\":\"e4\"}"],
					["1785585703000000000", "{\"action\":\"e3\"}"],
					["1785585702000000000", "{\"action\":\"e2\"}"],
					["1785585701000000000", "{\"action\":\"e1\"}"]
				]}
			]}
		}`))
	}))
	defer srv.Close()

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))

	page2, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{Page: 2, Size: 2})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
	if page2.Total != 5 {
		t.Fatalf("expected total=5, got %d", page2.Total)
	}
	if len(page2.Entries) != 2 || page2.Entries[0].Action != "e3" || page2.Entries[1].Action != "e2" {
		t.Fatalf("unexpected page 2 entries: %+v", page2.Entries)
	}

	page3, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{Page: 3, Size: 2})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
	if len(page3.Entries) != 1 || page3.Entries[0].Action != "e1" {
		t.Fatalf("unexpected page 3 (partial) entries: %+v", page3.Entries)
	}
}

func TestLokiSystemLogRepository_FindPaginated_TimeRange(t *testing.T) {
	var gotStart, gotEnd string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotStart = r.URL.Query().Get("start")
		gotEnd = r.URL.Query().Get("end")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
	}))
	defer srv.Close()

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

	repo := repository.NewLokiSystemLogRepository(lokiclient.New(srv.URL))
	_, err := repo.FindPaginated(context.Background(), domain.SystemLogListFilter{
		From: &from, To: &to, Page: 1, Size: 20,
	})
	if err != nil {
		t.Fatalf("FindPaginated returned error: %v", err)
	}
	if gotStart != "1782864000000000000" {
		t.Errorf("expected start to be From's unix nanos, got %s", gotStart)
	}
	if gotEnd != "1785542399000000000" {
		t.Errorf("expected end to be To's unix nanos, got %s", gotEnd)
	}
}
