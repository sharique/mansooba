package dto

// SettingsResponse is the flat representation of all global settings.
type SettingsResponse struct {
	OrganizationName string `json:"organization_name"`
	DateFormat       string `json:"date_format"`
	TimeFormat       string `json:"time_format"`
	Locale           string `json:"locale"`
	WeekStartDay     string `json:"week_start_day"`
	// SystemLogRetentionDays is a string (not an int) for the same reason
	// every other setting value is a string: global_settings is a generic
	// key-value store (011-system-logs, FR-010).
	SystemLogRetentionDays string `json:"system_log_retention_days"`
	// DemoBannerEnabled is "true"/"false" as a string, same reason as above
	// (013-demo-instance-banner, FR-001).
	DemoBannerEnabled string `json:"demo_banner_enabled"`
	DemoBannerMessage string `json:"demo_banner_message"`
}

// PatchSettingsRequest allows partial updates — only keys present in the payload are changed.
type PatchSettingsRequest struct {
	OrganizationName       *string `json:"organization_name"`
	DateFormat             *string `json:"date_format"`
	TimeFormat             *string `json:"time_format"`
	Locale                 *string `json:"locale"`
	WeekStartDay           *string `json:"week_start_day"`
	SystemLogRetentionDays *string `json:"system_log_retention_days"`
	DemoBannerEnabled      *string `json:"demo_banner_enabled"`
	DemoBannerMessage      *string `json:"demo_banner_message"`
}
