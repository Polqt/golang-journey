// Package cmd provides the CLI for the time-series database.
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Polqt/tsdb/tsdb"
)

// Run dispatches CLI subcommands.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "write":
		return runWrite(args[1:])
	case "query":
		return runQuery(args[1:])
	case "serve":
		return runServe(args[1:])
	case "compact":
		return runCompact(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println(`tsdb — embedded time-series database

Commands:
  write  <dir> <labels> <ts_ms> <value>
         Write a single sample. Labels: foo=bar,baz=qux
  query  <dir> <selector> <from_ms> <to_ms>
         Query samples in a time range.
  serve  <dir> [addr]
         Start HTTP API (write/query endpoint).
  compact <dir>
         Manually trigger block compaction.`)
}

func parseLabels(s string) tsdb.Labels {
	var labels tsdb.Labels
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			labels = append(labels, tsdb.Label{Name: kv[0], Value: kv[1]})
		}
	}
	return labels
}

func openDB(dir string) (*tsdb.DB, error) {
	return tsdb.Open(tsdb.DefaultOptions(dir))
}

func runWrite(args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: write <dir> <labels> <ts_ms> <value>")
	}
	db, err := openDB(args[0])
	if err != nil {
		return err
	}
	defer db.Close()

	labels := parseLabels(args[1])
	ts, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return fmt.Errorf("bad ts: %w", err)
	}
	val, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return fmt.Errorf("bad value: %w", err)
	}

	return db.Write(labels, []tsdb.Sample{{Ts: ts, Val: val}})
}

func runQuery(args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: query <dir> <selector> <from_ms> <to_ms>")
	}
	db, err := openDB(args[0])
	if err != nil {
		return err
	}
	defer db.Close()

	sel := parseLabels(args[1])
	from, _ := strconv.ParseInt(args[2], 10, 64)
	to, _ := strconv.ParseInt(args[3], 10, 64)

	series, err := db.Query(sel, from, to)
	if err != nil {
		return err
	}
	for _, s := range series {
		samples := s.Samples(from, to)
		fmt.Printf("%s  (%d samples)\n", s.Labels, len(samples))
		for _, sm := range samples {
			fmt.Printf("  %d  %f\n", sm.Ts, sm.Val)
		}
	}
	return nil
}

func runServe(args []string) error {
	dir := "./data"
	addr := ":9000"
	if len(args) > 0 {
		dir = args[0]
	}
	if len(args) > 1 {
		addr = args[1]
	}
	db, err := openDB(dir)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	// POST /write  body: {"labels":{"job":"api"},"samples":[{"ts":1234,"val":3.14}]}
	mux.HandleFunc("POST /write", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Labels  map[string]string `json:"labels"`
			Samples []struct {
				Ts  int64   `json:"ts"`
				Val float64 `json:"val"`
			} `json:"samples"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var labels tsdb.Labels
		for k, v := range req.Labels {
			labels = append(labels, tsdb.Label{Name: k, Value: v})
		}
		var samples []tsdb.Sample
		for _, s := range req.Samples {
			samples = append(samples, tsdb.Sample{Ts: s.Ts, Val: s.Val})
		}
		if err := db.Write(labels, samples); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// GET /query?selector=job%3Dapi&from=0&to=9999999999999
	mux.HandleFunc("GET /query", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		sel := parseLabels(q.Get("selector"))
		from, _ := strconv.ParseInt(q.Get("from"), 10, 64)
		to, _ := strconv.ParseInt(q.Get("to"), 10, 64)
		if to == 0 {
			to = time.Now().UnixMilli()
		}

		series, err := db.Query(sel, from, to)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		type respSeries struct {
			Labels  tsdb.Labels   `json:"labels"`
			Samples []tsdb.Sample `json:"samples"`
		}
		var resp []respSeries
		for _, s := range series {
			resp = append(resp, respSeries{Labels: s.Labels, Samples: s.Samples(from, to)})
			// s is *tsdb.Series — no mutex copy
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	fmt.Printf("tsdb HTTP API on %s, data dir %s\n", addr, dir)
	return http.ListenAndServe(addr, mux)
}

func runCompact(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: compact <dir>")
	}
	// TODO: load all head chunks, write them to a sealed block on disk,
	// and remove WAL segments that are now persisted.
	_ = args[0]
	fmt.Fprintf(os.Stderr, "compact: not yet implemented\n")
	return nil
}
