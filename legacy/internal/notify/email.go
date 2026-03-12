package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// EmailChannel sends alerts via SMTP with STARTTLS.
type EmailChannel struct {
	host     string
	port     int
	from     string
	to       []string
	username string
	password string
	enabled  bool
}

// NewEmailChannel creates an email notification channel.
func NewEmailChannel(host string, port int, from string, to []string, username, password string, enabled bool) *EmailChannel {
	return &EmailChannel{
		host:     host,
		port:     port,
		from:     from,
		to:       to,
		username: username,
		password: password,
		enabled:  enabled,
	}
}

// Name returns "email".
func (e *EmailChannel) Name() string { return "email" }

// Enabled returns whether this channel is active.
func (e *EmailChannel) Enabled() bool {
	return e.enabled && e.host != "" && len(e.to) > 0 && e.from != ""
}

// Send delivers an alert via SMTP with HTML body.
func (e *EmailChannel) Send(ctx context.Context, alert Alert) error {
	if e.host == "" || len(e.to) == 0 {
		return fmt.Errorf("email not configured")
	}

	emoji := SeverityEmoji(alert.Severity)
	subject := fmt.Sprintf("%s SENTINEL %s — %s", emoji, strings.ToUpper(alert.Severity), alert.Title)

	htmlBody := buildEmailHTML(alert)
	toList := strings.Join(e.to, ", ")

	msg := fmt.Sprintf(
		"From: SENTINEL <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		e.from, toList, subject, htmlBody,
	)

	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	// Use a channel to respect context cancellation with blocking SMTP calls.
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.sendSMTP(addr, msg)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *EmailChannel) sendSMTP(addr, msg string) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("email: dial %s: %w", addr, err)
	}
	defer c.Close()

	// STARTTLS
	tlsConfig := &tls.Config{ServerName: e.host}
	if err := c.StartTLS(tlsConfig); err != nil {
		// Some servers may not support STARTTLS — log but continue.
		_ = err
	}

	// Auth if credentials provided
	if e.username != "" && e.password != "" {
		auth := smtp.PlainAuth("", e.username, e.password, e.host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("email: auth: %w", err)
		}
	}

	if err := c.Mail(e.from); err != nil {
		return fmt.Errorf("email: MAIL FROM: %w", err)
	}
	for _, rcpt := range e.to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("email: RCPT TO %s: %w", rcpt, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("email: DATA: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("email: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email: close data: %w", err)
	}

	return c.Quit()
}

func buildEmailHTML(alert Alert) string {
	emoji := SeverityEmoji(alert.Severity)
	ts := alert.OccurredAt.Format(time.RFC1123)
	if alert.OccurredAt.IsZero() {
		ts = time.Now().UTC().Format(time.RFC1123)
	}

	var urlBlock string
	if alert.URL != "" {
		urlBlock = fmt.Sprintf(`<p><a href="%s" style="color:#00ccff;">View Event &rarr;</a></p>`, alert.URL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#0a0a0a;color:#e0e0e0;font-family:monospace,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#0a0a0a;">
    <tr><td align="center" style="padding:24px 0;">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#1a1a2e;border:1px solid #333;border-radius:8px;">
        <tr><td style="padding:20px 24px;background:#16213e;border-radius:8px 8px 0 0;">
          <h1 style="margin:0;color:#00ccff;font-size:18px;">&#128752; SENTINEL</h1>
        </td></tr>
        <tr><td style="padding:20px 24px;">
          <p style="margin:0 0 12px;font-size:14px;">%s <strong style="color:#00ccff;">%s</strong></p>
          <h2 style="margin:0 0 12px;color:#ffffff;font-size:16px;">%s</h2>
          <p style="margin:0 0 12px;color:#cccccc;font-size:13px;">%s</p>
          <p style="margin:0 0 8px;color:#888;font-size:12px;">Category: %s</p>
          <p style="margin:0 0 8px;color:#888;font-size:12px;">Time: %s</p>
          %s
        </td></tr>
        <tr><td style="padding:12px 24px;background:#0f0f23;border-radius:0 0 8px 8px;text-align:center;">
          <p style="margin:0;color:#555;font-size:11px;">SENTINEL World Monitoring System</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`,
		emoji, strings.ToUpper(alert.Severity),
		alert.Title, alert.Body,
		alert.Category, ts,
		urlBlock,
	)
}

// Test sends a test email.
func (e *EmailChannel) Test(ctx context.Context) error {
	return e.Send(ctx, Alert{
		Title:    "SENTINEL notification test",
		Body:     "If you received this email, SENTINEL email notifications are working correctly.",
		Severity: "info",
		Category: "test",
	})
}
