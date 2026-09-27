package service

import (
	"testing"

	"github.com/sharique/mansooba/internal/domain"
)

func TestInstanceService_Info(t *testing.T) {
	t.Run("returns exactly the injected version and source URL", func(t *testing.T) {
		svc := NewInstanceService(domain.InstanceInfo{
			Version:   "1.4.0",
			SourceURL: "https://example.com/my-fork",
		})
		got := svc.Info()
		if got.Version != "1.4.0" {
			t.Fatalf("Version = %q, want %q", got.Version, "1.4.0")
		}
		if got.SourceURL != "https://example.com/my-fork" {
			t.Fatalf("SourceURL = %q, want %q", got.SourceURL, "https://example.com/my-fork")
		}
	})

	t.Run("license and license URL are the fixed constants", func(t *testing.T) {
		svc := NewInstanceService(domain.InstanceInfo{Version: "1.0.0", SourceURL: "https://example.com"})
		got := svc.Info()
		if got.License != "AGPL-3.0-only" {
			t.Fatalf("License = %q, want %q", got.License, "AGPL-3.0-only")
		}
		if got.LicenseURL != "https://www.gnu.org/licenses/agpl-3.0.html" {
			t.Fatalf("LicenseURL = %q, want %q", got.LicenseURL, "https://www.gnu.org/licenses/agpl-3.0.html")
		}
	})

	t.Run("an empty version becomes dev", func(t *testing.T) {
		svc := NewInstanceService(domain.InstanceInfo{Version: "", SourceURL: "https://example.com"})
		got := svc.Info()
		if got.Version != "dev" {
			t.Fatalf("Version = %q, want %q", got.Version, "dev")
		}
	})
}
