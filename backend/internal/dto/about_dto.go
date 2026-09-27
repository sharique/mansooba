package dto

// AboutResponse is returned by GET /api/v1/about.
type AboutResponse struct {
	Version    string `json:"version"`
	SourceURL  string `json:"source_url"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
}
