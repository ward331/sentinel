package notify

import (
	"fmt"
)

// EmailChannel sends alerts via SMTP.
type EmailChannel struct {
	host     string
	port     int
	from     string
	to       []string
	username string
	password string
}

// NewEmailChannel creates an email notification channel.
func NewEmailChannel(host string, port int, from string, to []string) *EmailChannel {
	return &EmailChannel{host: host, port: port, from: from, to: to}
}

// Name returns "email".
func (e *EmailChannel) Name() string { return "email" }

// Send delivers an alert via SMTP.
// Stub — actual net/smtp call in Stage G5.
func (e *EmailChannel) Send(alert Alert) error {
	if e.host == "" || len(e.to) == 0 {
		return fmt.Errorf("email not configured")
	}
	// TODO: dial SMTP, send message
	return nil
}
