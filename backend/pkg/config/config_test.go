package config

import "testing"

// SOURCE_CODE_URL must be settable by an operator with no code change, and
// must be validated so the frontend never renders an unsafe link.
func TestValidateSourceCodeURL(t *testing.T) {
	const defaultURL = "https://github.com/sharique/mansooba"

	valid := []struct {
		name string
		raw  string
		want string
	}{
		{"unset gives the default", "", defaultURL},
		{"a valid https URL is kept", "https://example.com/fork", "https://example.com/fork"},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateSourceCodeURL(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	invalid := []struct {
		name string
		raw  string
	}{
		{"a javascript: URL is rejected", "javascript:alert(1)"},
		{"a non-http(s) scheme is rejected", "ftp://example.com"},
		{"a string with no scheme is rejected", "not a url"},
		{"embedded credentials are rejected", "https://user:pass@example.com/x"},
		{"a value over 2048 characters is rejected", "https://example.com/" + string(make([]byte, 2048))},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ValidateSourceCodeURL(tc.raw); err == nil {
				t.Fatalf("expected an error for %q, got none", tc.raw)
			}
		})
	}

	t.Run("an empty variable is treated the same as unset", func(t *testing.T) {
		got, err := ValidateSourceCodeURL("   ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != defaultURL {
			t.Fatalf("got %q, want the default %q", got, defaultURL)
		}
	})
}
