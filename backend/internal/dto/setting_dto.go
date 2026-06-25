package dto

// SettingsResponse is the flat representation of all five global settings.
type SettingsResponse struct {
	OrganizationName string `json:"organization_name"`
	DateFormat       string `json:"date_format"`
	TimeFormat       string `json:"time_format"`
	Locale           string `json:"locale"`
	WeekStartDay     string `json:"week_start_day"`
}

// PatchSettingsRequest allows partial updates — only keys present in the payload are changed.
type PatchSettingsRequest struct {
	OrganizationName *string `json:"organization_name"`
	DateFormat       *string `json:"date_format"`
	TimeFormat       *string `json:"time_format"`
	Locale           *string `json:"locale"`
	WeekStartDay     *string `json:"week_start_day"`
}
