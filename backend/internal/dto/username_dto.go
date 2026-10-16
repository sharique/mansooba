package dto

// UsernameCheckResponse is returned by GET /api/v1/admin/usernames/check.
// Reason is "" when the name is available, otherwise "invalid", "reserved" or
// "taken"; Message is text for the person filling in the form.
type UsernameCheckResponse struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}
