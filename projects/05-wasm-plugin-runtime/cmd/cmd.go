// Package cmd wires the WASM plugin runtime CLI.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"github.com/Polqt/wasmruntime/runtime"
)

// Run dispatches CLI subcommands.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "load":
		return runLoad(args[1:])
	case "call":
		return runCall(args[1:])
	case "serve":
		return runServe(args[1:])
	case "list":
		return runList()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println(`wasmruntime — WebAssembly plugin runtime

Commands:
  load  <id> <path.wasm>   Load and register a plugin
  call  <id> <func> <json> Call an exported function
  serve [addr]             Start the HTTP management API
  list                     List loaded plugins`)
}

// Global engine (single-process demo).
var eng = runtime.NewEngine(nil)

func runLoad(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: load <id> <path.wasm>")
	}
	id, path := args[0], args[1]
	cfg := runtime.DefaultPluginConfig()
	if err := eng.Load(id, path, cfg); err != nil {
		return fmt.Errorf("load: %w", err)
	}
	fmt.Printf("loaded plugin %q from %s\n", id, path)
	return nil
}

func runCall(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: call <id> <func> <json-args>")
	}
	id, fn, argsJSON := args[0], args[1], args[2]
	p, err := eng.Get(id)
	if err != nil {
		return err
	}
	result, err := p.Call(context.Background(), fn, []byte(argsJSON))
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	return nil
}

func runServe(args []string) error {
	addr := ":8080"
	if len(args) > 0 {
		addr = args[0]
	}

	mux := http.NewServeMux()

	// GET /plugins — list loaded plugins
	mux.HandleFunc("GET /plugins", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(eng.List())
	})

	// POST /plugins/{id}/load — load a plugin from body {"path":"..."}
	mux.HandleFunc("POST /plugins/{id}/load", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/plugins/")
		id = strings.TrimSuffix(id, "/load")
		var req struct{ Path string `json:"path"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		cfg := runtime.DefaultPluginConfig()
		if err := eng.Load(id, req.Path, cfg); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	// POST /plugins/{id}/call/{fn} — call exported func, body is json args
	mux.HandleFunc("POST /plugins/", func(w http.ResponseWriter, r *http.Request) {
		// Path: /plugins/{id}/call/{fn}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/plugins/"), "/")
		if len(parts) < 3 || parts[1] != "call" {
			http.NotFound(w, r)
			return
		}
		id, fn := parts[0], parts[2]
		p, err := eng.Get(id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		body, _ := os.ReadFile("/dev/stdin") // placeholder
		_ = body
		result, err := p.Call(r.Context(), fn, []byte("{}"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	})

	fmt.Printf("wasmruntime API listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func runList() error {
	ids := eng.List()
	if len(ids) == 0 {
		fmt.Println("(no plugins loaded)")
		return nil
	}
	for _, id := range ids {
		p, _ := eng.Get(id)
		fmt.Printf("  %-20s  v%s  exports=%v\n", id, p.Meta.Version, p.Meta.Exports)
	}
	return nil
}
