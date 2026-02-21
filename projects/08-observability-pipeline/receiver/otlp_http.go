// Package receiver implements an HTTP endpoint that accepts OTLP/JSON traces.
// It parses the request into pipeline.Batch items and forwards them to the channel.
package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Polqt/obspipeline/pipeline"
)

// ─────────────────────────────────────────────────────────────
// OTLP HTTP Receiver
// ─────────────────────────────────────────────────────────────

// OTLPHTTPReceiver listens for OTLP/JSON POST requests on /v1/traces, /v1/metrics, /v1/logs.
type OTLPHTTPReceiver struct {
	addr   string
	server *http.Server
	out    chan<- *pipeline.Batch
	mu     sync.Mutex
}

// NewOTLPHTTP creates a new OTLP HTTP receiver that will listen on addr.
func NewOTLPHTTP(addr string) *OTLPHTTPReceiver {
	return &OTLPHTTPReceiver{addr: addr}
}

// Name identifies this receiver.
func (r *OTLPHTTPReceiver) Name() string { return "otlp-http:" + r.addr }

// Start begins accepting connections and forwarding batches.
func (r *OTLPHTTPReceiver) Start(ctx context.Context, out chan<- *pipeline.Batch) error {
	r.mu.Lock()
	r.out = out
	r.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/traces", r.handleTraces)
	mux.HandleFunc("POST /v1/metrics", r.handleMetrics)
	mux.HandleFunc("POST /v1/logs", r.handleLogs)

	r.server = &http.Server{
		Addr:        r.addr,
		Handler:     mux,
		ReadTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("OTLP HTTP receiver started", "addr", r.addr)
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("OTLP HTTP receiver error", "err", err)
		}
	}()
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (r *OTLPHTTPReceiver) Stop() error {
	if r.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.server.Shutdown(ctx)
}

// ─── Trace handler ───

// otlpTracesReq is a minimal subset of the OTLP JSON trace request struct.
type otlpTracesReq struct {
	ResourceSpans []struct {
		ScopeSpans []struct {
			Spans []struct {
				TraceID    string `json:"traceId"` // hex
				SpanID     string `json:"spanId"`
				ParentSpanID string `json:"parentSpanId"`
				Name       string `json:"name"`
				Kind       int    `json:"kind"`
				StartTimeUnixNano string `json:"startTimeUnixNano"`
				EndTimeUnixNano   string `json:"endTimeUnixNano"`
				Status struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"status"`
			} `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}

func (r *OTLPHTTPReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	var otlpReq otlpTracesReq
	if err := json.NewDecoder(req.Body).Decode(&otlpReq); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var spans []*pipeline.Span
	for _, rs := range otlpReq.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				span := &pipeline.Span{
					Name:       s.Name,
					Kind:       pipeline.SpanKind(s.Kind),
					StatusCode: s.Status.Code,
					StatusMsg:  s.Status.Message,
				}
				// TODO: parse hex TraceID/SpanID/ParentID into [16]byte / [8]byte
				// TODO: parse StartTimeUnixNano / EndTimeUnixNano (string uint64)
				spans = append(spans, span)
			}
		}
	}

	if len(spans) > 0 {
		r.out <- &pipeline.Batch{Kind: pipeline.Traces, Spans: spans}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"partialSuccess":{}}`)
}

func (r *OTLPHTTPReceiver) handleMetrics(w http.ResponseWriter, req *http.Request) {
	// TODO: parse OTLP metrics JSON into []*pipeline.DataPoint
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"partialSuccess":{}}`)
}

func (r *OTLPHTTPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	// TODO: parse OTLP logs JSON into []*pipeline.LogRecord
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"partialSuccess":{}}`)
}
