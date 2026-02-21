// Package session manages connected WebSocket clients and message routing.
package session

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/Polqt/crdtcollab/crdt"
)

// ─────────────────────────────────────────────────────────────
// Message types
// ─────────────────────────────────────────────────────────────

const (
	MsgInsert   = "insert"
	MsgDelete   = "delete"
	MsgSnapshot = "snapshot"
	MsgAck      = "ack"
	MsgError    = "error"
)

// Message is the wire format for a CRDT operation.
type Message struct {
	DocID    string            `json:"doc_id"`
	Type     string            `json:"type"`
	Payload  json.RawMessage   `json:"payload"`
	Clock    map[string]uint64 `json:"clock,omitempty"`
	SenderID string            `json:"sender_id"`
	Ts       time.Time         `json:"ts"`
}

// InsertPayload carries an RGA insert operation.
type InsertPayload struct {
	AfterID crdt.RGANodeID `json:"after_id"`
	Char    string         `json:"char"` // single rune as string
	NodeID  string         `json:"node_id"`
}

// DeletePayload carries an RGA delete operation.
type DeletePayload struct {
	ID crdt.RGANodeID `json:"id"`
}

// SnapshotPayload is sent to new joiners.
type SnapshotPayload struct {
	Text  string            `json:"text"`
	Clock map[string]uint64 `json:"clock"`
}

// ─────────────────────────────────────────────────────────────
// Session
// ─────────────────────────────────────────────────────────────

// Sender is implemented by the WebSocket transport layer so Session
// can push messages without depending on the transport package.
type Sender interface {
	Send(msg Message) error
	Close() error
	RemoteAddr() string
}

// Session represents one connected client editing a document.
type Session struct {
	ID     string // unique session id (UUID)
	DocID  string
	NodeID string // CRDT node identity
	sender Sender
	hub    *Hub
}

// NewSession creates a session with the given transport sender.
func NewSession(id, docID, nodeID string, sender Sender, hub *Hub) *Session {
	return &Session{ID: id, DocID: docID, NodeID: nodeID, sender: sender, hub: hub}
}

// Push sends a message to this client.
func (s *Session) Push(msg Message) error {
	return s.sender.Send(msg)
}

// ─────────────────────────────────────────────────────────────
// Document — per-document CRDT state + sessions
// ─────────────────────────────────────────────────────────────

// Document holds the live CRDT state for one collaborative document.
type Document struct {
	mu       sync.RWMutex
	ID       string
	rga      *crdt.RGA
	sessions map[string]*Session // sessionID → session
}

// NewDocument creates a new empty document.
func NewDocument(id string) *Document {
	return &Document{
		ID:       id,
		rga:      crdt.NewRGA(),
		sessions: make(map[string]*Session),
	}
}

// Text returns the current document text (read-only snapshot).
func (d *Document) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.rga.Text()
}

// Apply dispatches an incoming operation to the RGA and broadcasts it.
func (d *Document) Apply(op crdt.RGANode) error {
	d.mu.Lock()
	err := d.rga.Apply(op)
	d.mu.Unlock()
	return err
}

// Broadcast sends msg to every session except excludeID.
func (d *Document) Broadcast(msg Message, excludeID string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for id, s := range d.sessions {
		if id == excludeID {
			continue
		}
		if err := s.Push(msg); err != nil {
			slog.Warn("broadcast failed", "session", id, "err", err)
		}
	}
}

// ─────────────────────────────────────────────────────────────
// Hub — registry of all documents and sessions
// ─────────────────────────────────────────────────────────────

// Hub is the central message router for all active documents and sessions.
type Hub struct {
	mu   sync.RWMutex
	docs map[string]*Document // docID → document
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{docs: make(map[string]*Document)}
}

// Run is a no-op placeholder for background maintenance (e.g. idle-doc cleanup).
// Call as a goroutine: go hub.Run()
func (h *Hub) Run() {
	// TODO: periodically evict documents with zero active sessions to reclaim memory.
}

// GetOrCreate returns the document with the given id, creating it if needed.
func (h *Hub) GetOrCreate(docID string) *Document {
	h.mu.Lock()
	defer h.mu.Unlock()
	if d, ok := h.docs[docID]; ok {
		return d
	}
	d := NewDocument(docID)
	h.docs[docID] = d
	return d
}

// Join registers a session with its document and sends the current snapshot.
func (h *Hub) Join(sess *Session) {
	doc := h.GetOrCreate(sess.DocID)
	doc.mu.Lock()
	doc.sessions[sess.ID] = sess
	text := doc.rga.Text()
	doc.mu.Unlock()

	snap, _ := json.Marshal(SnapshotPayload{Text: text})
	_ = sess.Push(Message{
		DocID:   sess.DocID,
		Type:    MsgSnapshot,
		Payload: snap,
		Ts:      time.Now(),
	})
}

// Leave removes a session from its document.
func (h *Hub) Leave(sess *Session) {
	doc := h.GetOrCreate(sess.DocID)
	doc.mu.Lock()
	delete(doc.sessions, sess.ID)
	doc.mu.Unlock()

	slog.Info("session left", "session", sess.ID, "doc", sess.DocID)
}

// Dispatch handles an incoming message from a session.
func (h *Hub) Dispatch(sess *Session, msg Message) {
	doc := h.GetOrCreate(msg.DocID)

	switch msg.Type {
	case MsgInsert:
		var p InsertPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			slog.Warn("bad insert payload", "err", err)
			return
		}
		// TODO: call doc.rga.Insert(p.AfterID, rune(p.Char[0]), p.NodeID)
		// and broadcast the resulting RGANode back to all other sessions
		doc.Broadcast(msg, sess.ID)

	case MsgDelete:
		var p DeletePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			slog.Warn("bad delete payload", "err", err)
			return
		}
		// TODO: call doc.rga.Delete(p.ID) and broadcast
		doc.Broadcast(msg, sess.ID)

	default:
		slog.Warn("unknown message type", "type", msg.Type)
	}
}
