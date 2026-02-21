// Package transport provides the WebSocket upgrade handler.
package transport

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Polqt/crdtcollab/session"
)

// ─────────────────────────────────────────────────────────────
// Minimal WebSocket implementation (RFC 6455, stdlib-only)
// ─────────────────────────────────────────────────────────────

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsHandshake performs the HTTP→WebSocket upgrade.
// Returns the hijacked net.Conn and bufio.Reader on success.
func wsHandshake(w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.ReadWriter, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, nil, fmt.Errorf("not a websocket upgrade")
	}
	key := r.Header.Get("Sec-Websocket-Key")
	if key == "" {
		return nil, nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}

	// Compute accept key.
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack unsupported")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}

	// Write upgrade response directly.
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		conn.Close()
		return nil, nil, err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return nil, nil, err
	}
	return conn, rw, nil
}

// WSConn is a minimal WebSocket connection (text frames only).
type WSConn struct {
	conn net.Conn
	rw   *bufio.ReadWriter
	mu   sync.Mutex
}

// ReadMessage reads the next WebSocket text frame payload.
func (c *WSConn) ReadMessage() ([]byte, error) {
	// TODO: parse RFC 6455 frame header:
	//   byte 0: FIN(1) + RSV(3) + Opcode(4)
	//   byte 1: MASK(1) + PayloadLen(7)   (extended if 126/127)
	//   if MASK: 4-byte masking key, then XOR-decode payload
	// Return payload of text frames (opcode 0x1); handle close (0x8) and ping (0x9).
	panic("WSConn.ReadMessage: not yet implemented")
}

// WriteMessage sends a text frame with the given payload.
func (c *WSConn) WriteMessage(payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO: build RFC 6455 frame:
	//   byte 0: 0x81 (FIN=1, opcode=text)
	//   bytes 1+: payload length (7-bit or 16-bit or 64-bit), then payload
	// Server frames must NOT be masked (clients must be masked).
	panic("WSConn.WriteMessage: not yet implemented")
}

// Close sends a WebSocket close frame and closes the underlying conn.
func (c *WSConn) Close() error {
	// TODO: write opcode 0x8 close frame, then conn.Close()
	return c.conn.Close()
}

// RemoteAddr returns the remote address string.
func (c *WSConn) RemoteAddr() string { return c.conn.RemoteAddr().String() }

// ─────────────────────────────────────────────────────────────
// wsSender — adapts WSConn to session.Sender
// ─────────────────────────────────────────────────────────────

type wsSender struct {
	ws *WSConn
}

func (s *wsSender) Send(msg session.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.ws.WriteMessage(b)
}

func (s *wsSender) Close() error       { return s.ws.Close() }
func (s *wsSender) RemoteAddr() string { return s.ws.RemoteAddr() }

// ─────────────────────────────────────────────────────────────
// WSHandler
// ─────────────────────────────────────────────────────────────

// WSHandler handles WebSocket upgrade requests and feeds messages to the Hub.
type WSHandler struct {
	hub *session.Hub
}

// NewWSHandler creates a handler backed by the given Hub.
func NewWSHandler(hub *session.Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// ServeHTTP upgrades the connection and starts the read loop.
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, rw, err := wsHandshake(w, r)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	ws := &WSConn{conn: conn, rw: rw}
	docID := strings.TrimPrefix(r.URL.Path, "/ws/")
	if docID == "" {
		docID = "default"
	}

	sess := &session.Session{
		ID:     fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano()),
		DocID:  docID,
		NodeID: conn.RemoteAddr().String(),
	}
	// Use reflection-free interface injection pattern:
	// We need to set the unexported sender field via a constructor.
	// Instead, use NewSession helper:
	sess2 := session.NewSession(sess.ID, docID, sess.NodeID, &wsSender{ws: ws}, h.hub)
	h.hub.Join(sess2)
	defer h.hub.Leave(sess2)

	for {
		payload, err := ws.ReadMessage()
		if err != nil {
			if err != io.EOF {
				slog.Warn("ws read error", "session", sess2.ID, "err", err)
			}
			return
		}
		var msg session.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			slog.Warn("bad json", "err", err)
			continue
		}
		msg.DocID = docID
		h.hub.Dispatch(sess2, msg)
	}
}
