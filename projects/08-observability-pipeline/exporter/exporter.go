// Package exporter contains span/metric/log exporters.
package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Polqt/obspipeline/pipeline"
)

// ─────────────────────────────────────────────────────────────
// Stdout exporter (development / debugging)
// ─────────────────────────────────────────────────────────────

// StdoutExporter writes every batch as JSON to stdout.
type StdoutExporter struct {
	pretty bool
}

// NewStdout creates an exporter that prints batches to stdout.
func NewStdout(pretty bool) *StdoutExporter { return &StdoutExporter{pretty: pretty} }

func (*StdoutExporter) Name() string { return "stdout" }

func (e *StdoutExporter) Export(_ context.Context, b *pipeline.Batch) error {
	var data []byte
	var err error
	if e.pretty {
		data, err = json.MarshalIndent(b, "", "  ")
	} else {
		data, err = json.Marshal(b)
	}
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

// ─────────────────────────────────────────────────────────────
// OTLP/HTTP exporter (forward to a real backend)
// ─────────────────────────────────────────────────────────────

// OTLPHTTPExporter forwards batches to a downstream OTLP/HTTP endpoint.
type OTLPHTTPExporter struct {
	endpoint string      // e.g. http://otelcol:4318
	client   *http.Client
	headers  map[string]string
}

// NewOTLPHTTP creates an exporter for the given OTLP endpoint.
func NewOTLPHTTP(endpoint string, headers map[string]string) *OTLPHTTPExporter {
	return &OTLPHTTPExporter{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 10 * time.Second},
		headers:  headers,
	}
}

func (e *OTLPHTTPExporter) Name() string { return "otlp-http:" + e.endpoint }

func (e *OTLPHTTPExporter) Export(ctx context.Context, b *pipeline.Batch) error {
	var path string
	switch b.Kind {
	case pipeline.Traces:
		path = "/v1/traces"
	case pipeline.Metrics:
		path = "/v1/metrics"
	case pipeline.Logs:
		path = "/v1/logs"
	default:
		return fmt.Errorf("unknown signal kind %q", b.Kind)
	}

	// TODO: serialize to proper OTLP protobuf/JSON format.
	// For now, send our internal JSON representation.
	body, err := json.Marshal(b)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range e.headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("export POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("export POST %s: status %d", path, resp.StatusCode)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// Prometheus exporter (expose metrics on /metrics)
// ─────────────────────────────────────────────────────────────

// PrometheusExporter accumulates DataPoints and exposes them on a /metrics endpoint.
type PrometheusExporter struct {
	addr     string
	mu       struct{ _ [0]func() } // zero-size placeholder; use sync.Mutex in impl
	gauges   map[string]float64
}

// NewPrometheus creates a Prometheus exposition exporter listening on addr.
func NewPrometheus(addr string) *PrometheusExporter {
	return &PrometheusExporter{addr: addr, gauges: make(map[string]float64)}
}

func (e *PrometheusExporter) Name() string { return "prometheus:" + e.addr }

func (e *PrometheusExporter) Export(_ context.Context, b *pipeline.Batch) error {
	if b.Kind != pipeline.Metrics {
		return nil
	}
	// TODO: update e.gauges map from b.Points, then serve via /metrics handler
	// Format: "# TYPE <name> gauge\n<name>{labels} <value> <ts>\n"
	for _, dp := range b.Points {
		e.gauges[dp.Name] = dp.Value
	}
	return nil
}

// ServeHTTP serves the Prometheus text format.
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: write proper Prometheus text exposition format
	for name, val := range e.gauges {
		fmt.Fprintf(w, "# TYPE %s gauge\n%s %.6f\n", name, name, val)
	}
}
