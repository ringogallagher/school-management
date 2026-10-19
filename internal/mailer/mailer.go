// Package mailer sends notification e-mails. If SMTP is not configured the
// LogMailer is used, which only writes the message to the application log -
// useful for development and for the thesis demo without a real mail server.
package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Mailer sends a plain-text e-mail
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogMailer logs e-mails instead of sending them
type LogMailer struct{}

// Send writes the e-mail to the log
func (LogMailer) Send(ctx context.Context, to, subject, body string) error {
	slog.Info("email (log only - SMTP not configured)", "to", to, "subject", subject, "body", body)
	return nil
}

// SMTPConfig holds SMTP connection settings
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// SMTPMailer sends e-mail through an SMTP server using PLAIN auth (STARTTLS when offered)
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer creates an SMTP mailer
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return &SMTPMailer{cfg: cfg}
}

// sanitizeHeader prevents header injection through CR/LF in user-controlled values
func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

// BuildMessage builds an RFC 5322 message; exported for tests
func BuildMessage(from, to, subject, body string, now time.Time) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", sanitizeHeader(from))
	fmt.Fprintf(&b, "To: %s\r\n", sanitizeHeader(to))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", sanitizeHeader(subject)))
	fmt.Fprintf(&b, "Date: %s\r\n", now.Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	b.WriteString("\r\n")
	return []byte(b.String())
}

// Send delivers the e-mail via SMTP
func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	msg := BuildMessage(m.cfg.From, to, subject, body, time.Now())

	done := make(chan error, 1)
	go func() { done <- smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg) }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("smtp send to %s failed: %w", to, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
