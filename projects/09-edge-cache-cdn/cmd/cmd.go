// Package cmd provides the CLI for the edge cache.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Polqt/edgecache/edge"
)

// Run dispatches CLI subcommands.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "purge":
		return runPurge(args[1:])
	case "version":
		fmt.Println("edgecache v0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println(`edgecache — HTTP caching edge proxy

Commands:
  serve  <origin-url> [listen-addr] [admin-addr]
         Start the caching proxy.
         Defaults: listen on :8080, admin on :9000.

  purge  <admin-addr> <url-prefix>
         Purge cached entries whose key starts with prefix.

  version`)
}

func runServe(args []string) error {
	cfg := edge.DefaultConfig()
	if len(args) > 0 {
		cfg.OriginURL = args[0]
	}
	if len(args) > 1 {
		cfg.ListenAddr = args[1]
	}
	if len(args) > 2 {
		cfg.AdminAddr = args[2]
	}

	proxy, err := edge.NewProxy(cfg)
	if err != nil {
		return err
	}

	// Admin API (purge, stats, health).
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("POST /purge", func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			http.Error(w, "prefix required", 400)
			return
		}
		n := proxy.Purge(prefix)
		json.NewEncoder(w).Encode(map[string]int{"purged": n})
	})
	adminMux.HandleFunc("GET /stats", func(w http.ResponseWriter, _ *http.Request) {
		hits, misses, stales := proxy.Stats()
		size, cap := proxy.StoreStats()
		json.NewEncoder(w).Encode(map[string]any{
			"hits": hits, "misses": misses, "stales": stales,
			"cache_size": size, "cache_capacity": cap,
		})
	})
	adminMux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	proxyServer := &http.Server{Addr: cfg.ListenAddr, Handler: proxy}
	adminServer := &http.Server{Addr: cfg.AdminAddr, Handler: adminMux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("edgecache proxy on %s → %s\n", cfg.ListenAddr, cfg.OriginURL)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "proxy error:", err)
		}
	}()
	go func() {
		fmt.Printf("edgecache admin on %s\n", cfg.AdminAddr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "admin error:", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nshutting down...")
	proxyServer.Shutdown(context.Background())
	adminServer.Shutdown(context.Background())
	return nil
}

func runPurge(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: purge <admin-addr> <url-prefix>")
	}
	addr, prefix := args[0], args[1]
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	resp, err := http.Post(addr+"/purge?prefix="+prefix, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("purged %d entries\n", result["purged"])
	return nil
}
