// Package cmd is the CLI entry point for meshproxy.
package cmd

import (
	"fmt"
	"net"

	"github.com/Polqt/meshproxy/config"
	"github.com/Polqt/meshproxy/metrics"
	"github.com/Polqt/meshproxy/proxy"
)

// Run parses args and starts the sidecar proxy.
func Run(args []string) error {
	// Subcommands: proxy, inspect, version
	if len(args) == 0 {
		return runProxy()
	}
	switch args[0] {
	case "proxy":
		return runProxy()
	case "inspect":
		return runInspect()
	case "version":
		fmt.Println("meshproxy v0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command %q — try: proxy, inspect, version", args[0])
	}
}

func runProxy() error {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	col := metrics.NewCollector()

	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}

	srv := proxy.NewServer(cfg, col)
	fmt.Printf("meshproxy listening on %s → upstreams %v\n", cfg.ListenAddr, cfg.Upstreams)
	return srv.Serve(listener)
}

func runInspect() error {
	// TODO: fetch and pretty-print metrics from the admin HTTP server
	return fmt.Errorf("inspect: not yet implemented")
}
