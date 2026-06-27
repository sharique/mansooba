package email

import (
	"context"
	"fmt"
	"net/smtp"
)

// SMTPSender delivers password-reset messages via plain unauthenticated SMTP.
// Suitable for local dev (Mailpit) and internal relay servers that don't
// require auth. For authenticated delivery (Gmail, SES SMTP) extend this
// struct with Username/Password fields and pass smtp.PlainAuth.
type SMTPSender struct {
	Host string // e.g. "mailpit" or "smtp.example.com"
	Port string // e.g. "1025" (Mailpit) or "587" (TLS submission)
	From string // e.g. "noreply@mansooba.local"
}

func (s SMTPSender) SendPasswordReset(_ context.Context, to, token string) error {
	addr := s.Host + ":" + s.Port
	body := fmt.Sprintf(
		"You requested a password reset for your Mansooba account.\r\n\r\n"+
			"Your reset token is:\r\n\r\n  %s\r\n\r\n"+
			"This token expires in 15 minutes.\r\n"+
			"If you did not request this, you can safely ignore this message.\r\n",
		token,
	)
	msg := []byte(
		"To: " + to + "\r\n" +
			"From: " + s.From + "\r\n" +
			"Subject: Reset your Mansooba password\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"\r\n" +
			body,
	)
	return smtp.SendMail(addr, nil, s.From, []string{to}, msg)
}
