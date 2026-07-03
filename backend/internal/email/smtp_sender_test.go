package email

import (
	"context"
	"net/smtp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureSender builds an SMTPSender that captures the outbound message bytes
// instead of dialling a real SMTP server.
func captureSender(baseURL string) (SMTPSender, *[]byte) {
	var captured []byte
	s := SMTPSender{
		Host:       "localhost",
		Port:       "1025",
		From:       "test@example.com",
		AppBaseURL: baseURL,
		sendMailFn: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			captured = append([]byte(nil), msg...) // copy to avoid mutation
			return nil
		},
	}
	return s, &captured
}

// TestSendPasswordReset_WithBaseURL_HasMagicLink verifies that when AppBaseURL
// is set the outbound message is multipart/alternative and the HTML part
// contains a clickable anchor pointing to the reset page.
func TestSendPasswordReset_WithBaseURL_HasMagicLink(t *testing.T) {
	s, captured := captureSender("http://example.com")
	err := s.SendPasswordReset(context.Background(), "user@test.com", "abc123")
	require.NoError(t, err)

	body := string(*captured)
	assert.Contains(t, body, "multipart/alternative", "Content-Type header must declare multipart/alternative")
	assert.Contains(t, body, `href="http://example.com/reset-password?token=abc123"`, "HTML part must contain the magic link anchor")
	assert.Contains(t, body, "text/html", "message must have a text/html MIME part")
	assert.Contains(t, body, "text/plain", "message must retain the text/plain MIME part")
}

// TestSendPasswordReset_EmptyBaseURL_PlainTextOnly verifies that when
// AppBaseURL is empty the message falls back to plain-text only and emits
// no HTML anchor tag.
func TestSendPasswordReset_EmptyBaseURL_PlainTextOnly(t *testing.T) {
	s, captured := captureSender("")
	err := s.SendPasswordReset(context.Background(), "user@test.com", "abc123")
	require.NoError(t, err)

	body := string(*captured)
	assert.NotContains(t, body, "<a href", "plain-text fallback must not contain HTML anchor")
	assert.NotContains(t, body, "multipart/alternative", "plain-text fallback must not be multipart")
	assert.Contains(t, body, "abc123", "raw token must appear in the plain-text body")
}

// TestSendPasswordReset_URLUnsafeToken_IsEncoded verifies that a token
// containing URL-unsafe characters is encoded with url.QueryEscape so the
// resulting link is a valid URL.
func TestSendPasswordReset_URLUnsafeToken_IsEncoded(t *testing.T) {
	// "tok en+val/ue" → url.QueryEscape → "tok+en%2Bval%2Fue"
	s, captured := captureSender("http://example.com")
	err := s.SendPasswordReset(context.Background(), "user@test.com", "tok en+val/ue")
	require.NoError(t, err)

	body := string(*captured)
	assert.True(t,
		strings.Contains(body, "tok+en%2Bval%2Fue"),
		"token in URL must be url.QueryEscape-encoded; got body: %s", body,
	)
	// The href must not contain unencoded unsafe characters (raw token may appear in body text).
	assert.NotContains(t, body, `href="http://example.com/reset-password?token=tok en+val/ue"`,
		"href attribute must not contain raw unsafe characters")
}

// TestSendPasswordReset_TrailingSlash_NoDoubleSlash verifies that a trailing
// slash on AppBaseURL does not produce a double-slash in the magic link.
func TestSendPasswordReset_TrailingSlash_NoDoubleSlash(t *testing.T) {
	s, captured := captureSender("http://example.com/")
	err := s.SendPasswordReset(context.Background(), "user@test.com", "abc123")
	require.NoError(t, err)

	body := string(*captured)
	assert.NotContains(t, body, "//reset-password", "trailing slash must not produce double-slash in link")
	assert.Contains(t, body, "/reset-password?token=abc123", "link must have a single-slash path")
}

// TestSendPasswordReset_TokenValueUnmodified verifies that the token passed
// to SendPasswordReset appears verbatim in the plain-text body, confirming
// that the function does not mutate or truncate the token value (FR-006).
func TestSendPasswordReset_TokenValueUnmodified(t *testing.T) {
	const token = "original-token-value-unchanged"
	s, captured := captureSender("http://example.com")
	err := s.SendPasswordReset(context.Background(), "user@test.com", token)
	require.NoError(t, err)

	body := string(*captured)
	assert.Contains(t, body, token, "original token value must appear verbatim in the plain-text body")
}
