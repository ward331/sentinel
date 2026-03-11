package api

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	// RFC 6455 magic GUID for WebSocket handshake
	wsMagicGUID = "258EAFA5-E914-47DA-95CA-5AB5FFC11AD3"

	// WebSocket opcodes
	wsOpText  = 0x1
	wsOpClose = 0x8
	wsOpPing  = 0x9
	wsOpPong  = 0xA

	// Keepalive interval
	wsPingInterval = 30 * time.Second
)

// upgradeWebSocket performs the WebSocket upgrade handshake and returns
// the hijacked connection. Returns an error if the request is not a
// valid WebSocket upgrade or if hijacking fails.
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	// Validate Upgrade header
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, fmt.Errorf("missing or invalid Upgrade header")
	}
	if !headerContains(r.Header.Get("Connection"), "Upgrade") {
		return nil, fmt.Errorf("missing or invalid Connection header")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, fmt.Errorf("missing Sec-WebSocket-Key header")
	}

	// Compute accept key per RFC 6455 section 4.2.2
	acceptKey := computeAcceptKey(key)

	// Hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("response writer does not support hijacking")
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %w", err)
	}

	// Write the 101 Switching Protocols response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n" +
		"\r\n"

	if _, err := bufrw.WriteString(response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write upgrade response: %w", err)
	}
	if err := bufrw.Flush(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to flush upgrade response: %w", err)
	}

	return conn, nil
}

// computeAcceptKey computes the Sec-WebSocket-Accept value per RFC 6455.
func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsMagicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// headerContains checks whether a comma-separated header value contains
// the given token (case-insensitive).
func headerContains(header, token string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

// writeWSTextFrame writes a WebSocket text frame (server-to-client,
// unmasked per RFC 6455 section 5.1).
func writeWSTextFrame(conn net.Conn, data []byte) error {
	length := len(data)

	// Frame header: FIN=1, opcode=text (0x81), then payload length
	var header []byte
	if length <= 125 {
		header = []byte{0x81, byte(length)}
	} else if length <= 65535 {
		header = make([]byte, 4)
		header[0] = 0x81
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:4], uint16(length))
	} else {
		header = make([]byte, 10)
		header[0] = 0x81
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:10], uint64(length))
	}

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if _, err := conn.Write(data); err != nil {
		return err
	}
	return nil
}

// writeWSPingFrame sends a WebSocket ping frame.
func writeWSPingFrame(conn net.Conn) error {
	// FIN=1, opcode=ping (0x89), length=0
	_, err := conn.Write([]byte{0x89, 0x00})
	return err
}

// writeWSCloseFrame sends a WebSocket close frame with a status code.
func writeWSCloseFrame(conn net.Conn, code uint16) error {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, code)
	frame := []byte{0x88, 0x02} // FIN=1, opcode=close, length=2
	frame = append(frame, payload...)
	_, err := conn.Write(frame)
	return err
}

// readWSFrame reads a single WebSocket frame and returns the opcode and
// unmasked payload. It handles masked frames from clients per RFC 6455.
func readWSFrame(conn net.Conn) (opcode byte, payload []byte, err error) {
	// Read first 2 bytes
	head := make([]byte, 2)
	if _, err = io.ReadFull(conn, head); err != nil {
		return 0, nil, err
	}

	opcode = head[0] & 0x0F
	masked := (head[1] & 0x80) != 0
	length := uint64(head[1] & 0x7F)

	// Extended payload length
	switch length {
	case 126:
		ext := make([]byte, 2)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err = io.ReadFull(conn, ext); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext)
	}

	// Limit frame size to 64KB to prevent abuse
	if length > 65536 {
		return 0, nil, fmt.Errorf("frame too large: %d bytes", length)
	}

	// Read masking key if present
	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(conn, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}

	// Read payload
	payload = make([]byte, length)
	if length > 0 {
		if _, err = io.ReadFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}

	// Unmask payload
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return opcode, payload, nil
}

// wsMessage is the envelope for WebSocket JSON messages.
type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// HandleWebSocket handles GET /api/ws — upgrades to WebSocket and streams
// events using the same StreamBroker as the SSE endpoint.
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed from %s: %v", r.RemoteAddr, err)
		http.Error(w, fmt.Sprintf("WebSocket upgrade failed: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("WebSocket: New connection from %s", r.RemoteAddr)

	// Subscribe to the same stream broker used by SSE
	client := h.stream.NewClient()

	// Ensure cleanup on exit
	defer func() {
		h.stream.RemoveClient(client)
		writeWSCloseFrame(conn, 1000) // 1000 = normal closure
		conn.Close()
		log.Printf("WebSocket: Connection closed for %s", r.RemoteAddr)
	}()

	// Send a welcome message
	welcome, _ := json.Marshal(wsMessage{
		Type: "connected",
		Data: json.RawMessage(`{"message":"SENTINEL WebSocket stream active"}`),
	})
	if err := writeWSTextFrame(conn, welcome); err != nil {
		log.Printf("WebSocket: Failed to send welcome to %s: %v", r.RemoteAddr, err)
		return
	}

	// Channel to signal that the client has disconnected
	disconnected := make(chan struct{})

	// Goroutine: read frames from client (detect close/pong)
	go func() {
		defer close(disconnected)
		for {
			opcode, _, err := readWSFrame(conn)
			if err != nil {
				// Any read error means the client is gone
				return
			}
			switch opcode {
			case wsOpClose:
				return
			case wsOpPong:
				// Keepalive acknowledged — nothing to do
			case wsOpPing:
				// Client sent a ping; respond with pong
				conn.Write([]byte{0x8A, 0x00}) // FIN=1, opcode=pong, length=0
			}
		}
	}()

	// Ping ticker for keepalive
	pingTicker := time.NewTicker(wsPingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case event := <-client:
			// Marshal the event as the data payload
			eventData, err := json.Marshal(event)
			if err != nil {
				log.Printf("WebSocket: Failed to marshal event %s: %v", event.ID, err)
				continue
			}
			msg, _ := json.Marshal(wsMessage{
				Type: "new_event",
				Data: json.RawMessage(eventData),
			})
			if err := writeWSTextFrame(conn, msg); err != nil {
				log.Printf("WebSocket: Write failed for %s: %v", r.RemoteAddr, err)
				return
			}

		case <-pingTicker.C:
			if err := writeWSPingFrame(conn); err != nil {
				log.Printf("WebSocket: Ping failed for %s: %v", r.RemoteAddr, err)
				return
			}

		case <-disconnected:
			return

		case <-r.Context().Done():
			return
		}
	}
}
