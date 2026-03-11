package notify

import (
	"fmt"
)

// NtfyChannel sends alerts via ntfy.sh push notifications.
type NtfyChannel struct {
	server string
	topic  string
}

// NewNtfyChannel creates an ntfy notification channel.
func NewNtfyChannel(server, topic string) *NtfyChannel {
	if server == "" {
		server = "https://ntfy.sh"
	}
	return &NtfyChannel{server: server, topic: topic}
}

// Name returns "ntfy".
func (n *NtfyChannel) Name() string { return "ntfy" }

// Send delivers an alert via ntfy.
// Stub — actual HTTP POST in Stage G5.
func (n *NtfyChannel) Send(alert Alert) error {
	if n.topic == "" {
		return fmt.Errorf("ntfy topic not configured")
	}
	// TODO: POST to n.server/n.topic
	return nil
}
