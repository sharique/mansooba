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

	htmlBody := fmt.Sprintf(
		"<p>You requested a password reset for your Mansooba account.</p>"+
			"<p>Your reset token is: %s</p>"+
			`<p><a href="%s">Reset my password</a></p>`+
			"<p>This token expires in 15 minutes.</p>"+
			"<p>If you did not request this, you can safely ignore this message.</p>",
		token, magicLink,
	)

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
