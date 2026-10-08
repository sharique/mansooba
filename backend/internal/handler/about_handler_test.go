package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
)

// stubInstanceService is a controllable stand-in for service.InstanceService.
type stubInstanceService struct {
	info domain.InstanceInfo
}

func (s *stubInstanceService) Info() domain.InstanceInfo { return s.info }

// The about endpoint is public by design —
// no Authorization header is sent in any of these requests.
func TestAboutHandler_Get(t *testing.T) {
	svc := &stubInstanceService{info: domain.InstanceInfo{
		Version:    "1.4.0",
		SourceURL:  "https://github.com/sharique/mansooba",
		License:    "AGPL-3.0-only",
		LicenseURL: "https://www.gnu.org/licenses/agpl-3.0.html",
	}}
	e := newEcho()
	h := handler.NewAboutHandler(svc)
	e.GET("/about", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AboutResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode: %v", err)
	}
	if resp.Version != "1.4.0" {
		t.Errorf("version = %q, want %q", resp.Version, "1.4.0")
	}
	if resp.SourceURL != "https://github.com/sharique/mansooba" {
		t.Errorf("source_url = %q, want %q", resp.SourceURL, "https://github.com/sharique/mansooba")
	}
	if resp.License != "AGPL-3.0-only" {
		t.Errorf("license = %q, want %q", resp.License, "AGPL-3.0-only")
	}
	if resp.LicenseURL != "https://www.gnu.org/licenses/agpl-3.0.html" {
		t.Errorf("license_url = %q, want %q", resp.LicenseURL, "https://www.gnu.org/licenses/agpl-3.0.html")
	}

	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want %q", cc, "public, max-age=300")
	}
}

func TestAboutHandler_Get_RejectsPost(t *testing.T) {
	svc := &stubInstanceService{info: domain.InstanceInfo{Version: "1.0.0", SourceURL: "https://example.com"}}
	e := newEcho()
	h := handler.NewAboutHandler(svc)
	e.GET("/about", h.Get)

	req := httptest.NewRequest(http.MethodPost, "/about", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}
