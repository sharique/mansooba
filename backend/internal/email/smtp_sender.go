package email

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strings"
)

// SMTPSender delivers password-reset messages via plain unauthenticated SMTP.
// Suitable for local dev (Mailpit) and internal relay servers that don't
// require auth. For authenticated delivery (Gmail, SES SMTP) extend this
// struct with Username/Password fields and pass smtp.PlainAuth.
type SMTPSender struct {
	Host       string // e.g. "mailpit" or "smtp.example.com"
	Port       string // e.g. "1025" (Mailpit) or "587" (TLS submission)
	From       string // e.g. "noreply@mansooba.local"
	AppBaseURL string // e.g. "https://app.example.com"; empty = no magic link

	// sendMailFn is used only in tests to capture outbound bytes without a
	// real SMTP server. Nil in production — falls back to smtp.SendMail.
	sendMailFn func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

func (s SMTPSender) SendPasswordReset(_ context.Context, to, token string) error {
	addr := s.Host + ":" + s.Port

	plainBody := fmt.Sprintf(
		"You requested a password reset for your Mansooba account.\r\n\r\n"+
			"Your reset token is:\r\n\r\n  %s\r\n\r\n"+
			"This token expires in 15 minutes.\r\n"+
			"If you did not request this, you can safely ignore this message.\r\n",
		token,
	)

	send := s.sendMailFn
	if send == nil {
		send = smtp.SendMail
	}

	if s.AppBaseURL == "" {
		log.Printf("warning: APP_BASE_URL not set — sending raw token only")
		msg := []byte(
			"To: " + to + "\r\n" +
				"From: " + s.From + "\r\n" +
				"Subject: Reset your Mansooba password\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"\r\n" +
				plainBody,
		)
		return send(addr, nil, s.From, []string{to}, msg)
	}

	magicLink := strings.TrimRight(s.AppBaseURL, "/") +
		"/reset-password?token=" + url.QueryEscape(token)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<head><meta http-equiv="Content-Type" content="text/html; charset=UTF-8" /></head>
<body style="margin:0;padding:0;background-color:#f0f4f4;font-family:Arial,Helvetica,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f0f4f4;">
  <tr><td align="center" style="padding:40px 16px;">
    <table width="600" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;width:100%%;background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">
      <tr>
        <td style="background-color:#2a8080;padding:24px 40px;">
          <span style="color:#ffffff;font-size:20px;font-weight:700;letter-spacing:-0.3px;">Mansooba</span>
        </td>
      </tr>
      <tr>
        <td style="padding:40px 40px 32px;">
          <p style="margin:0 0 16px;font-size:16px;color:#2d3a40;line-height:1.6;">Hi there,</p>
          <p style="margin:0 0 28px;font-size:16px;color:#2d3a40;line-height:1.6;">We received a request to reset the password for your Mansooba account. Click the button below to choose a new password.</p>
          <table cellpadding="0" cellspacing="0" border="0" style="margin:0 0 32px;">
            <tr>
              <td style="background-color:#2a8080;border-radius:6px;">
                <a href="%s" style="display:inline-block;padding:14px 32px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;">Reset my password</a>
              </td>
            </tr>
          </table>
          <p style="margin:0 0 6px;font-size:13px;color:#6b7b80;line-height:1.5;">This link expires in <strong>15 minutes</strong>. If the button doesn't work, copy and paste this URL into your browser:</p>
          <p style="margin:0 0 32px;font-size:12px;color:#2a8080;word-break:break-all;">%s</p>
          <hr style="border:none;border-top:1px solid #e8eeee;margin:0 0 24px;" />
          <p style="margin:0;font-size:13px;color:#9aabaf;line-height:1.5;">If you didn't request a password reset, you can safely ignore this email — your password will remain unchanged.</p>
        </td>
      </tr>
      <tr>
        <td style="padding:14px 40px;background:#f8fbfb;border-top:1px solid #e8eeee;">
          <p style="margin:0;font-size:12px;color:#b0bec5;text-align:center;">Mansooba &middot; Project Management</p>
        </td>
      </tr>
    </table>
  </td></tr>
</table>
</body>
</html>`, magicLink, magicLink)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	headers := textproto.MIMEHeader{}
	headers.Set("To", to)
	headers.Set("From", s.From)
	headers.Set("Subject", "Reset your Mansooba password")
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", "multipart/alternative; boundary="+mw.Boundary())

	for k, vs := range headers {
		for _, v := range vs {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(&buf, "\r\n")

	// text/plain part — must come before text/html per RFC 2046 §5.1.4
	pw, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"7bit"},
	})
	fmt.Fprint(pw, plainBody)

	// text/html part
	hw, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=utf-8"},
		"Content-Transfer-Encoding": {"7bit"},
	})
	fmt.Fprint(hw, htmlBody)

	mw.Close()

	return send(addr, nil, s.From, []string{to}, buf.Bytes())
}
