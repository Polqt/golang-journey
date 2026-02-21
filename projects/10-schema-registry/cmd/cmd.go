// Package cmd provides the CLI for the schema registry.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	apiv1 "github.com/Polqt/schemaregistry/api/v1"
	"github.com/Polqt/schemaregistry/registry/compat"
	"github.com/Polqt/schemaregistry/registry/schema"
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
	case "register":
		return runRegister(args[1:])
	case "get":
		return runGet(args[1:])
	case "list":
		return runList(args[1:])
	case "compat":
		return runCompat(args[1:])
	case "diff":
		return runDiff(args[1:])
	case "version":
		fmt.Println("schemaregistry v0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println(`schemaregistry — schema registry with Confluent-compatible REST API

Commands:
  serve   [addr]                   Start the HTTP server (default :8081)
  register <subject> <file>        Register schema from file
  get     <subject> [version]      Get schema (default: latest)
  list    [subject]                List subjects or versions
  compat  <subject> <file>         Check if schema is compatible
  diff    <subject> <file>         Show diff vs latest schema
  version`)
}

// Global in-process registry (for serve command).
var reg = schema.New(&compat.Checker{})

func runServe(args []string) error {
	addr := ":8081"
	if len(args) > 0 {
		addr = args[0]
	}

	handler := apiv1.Handler(reg)
	server := &http.Server{Addr: addr, Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("schemaregistry listening on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "server error:", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nshutting down...")
	return server.Shutdown(context.Background())
}

func runRegister(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: register <subject> <file>")
	}
	subject, file := args[0], args[1]
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	format := detectFormat(file)
	id, err := reg.Register(subject, format, string(content))
	if err != nil {
		return err
	}
	fmt.Printf("registered %s as version %d (global id=%d)\n", subject, 1, id)
	return nil
}

func runGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get <subject> [version]")
	}
	subject := args[0]
	var s *schema.Schema
	var err error
	if len(args) == 1 {
		s, err = reg.GetLatest(subject)
	} else {
		var v int
		fmt.Sscanf(args[1], "%d", &v)
		s, err = reg.GetVersion(subject, v)
	}
	if err != nil {
		return err
	}
	fmt.Printf("subject=%s version=%d id=%d format=%s\n%s\n",
		s.Subject, s.Version, s.ID, s.Format, s.Content)
	return nil
}

func runList(args []string) error {
	if len(args) == 0 {
		for _, sub := range reg.Subjects() {
			fmt.Println(sub)
		}
		return nil
	}
	vers, err := reg.Versions(args[0])
	if err != nil {
		return err
	}
	for _, v := range vers {
		fmt.Println(v)
	}
	return nil
}

func runCompat(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: compat <subject> <file>")
	}
	subject, file := args[0], args[1]
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	latest, err := reg.GetLatest(subject)
	if err != nil {
		return err
	}
	level, _ := reg.CompatLevel(subject)
	checker := &compat.Checker{}
	if err := checker.Check(latest.Format, latest.Content, string(content), level); err != nil {
		fmt.Printf("INCOMPATIBLE: %v\n", err)
		return nil
	}
	fmt.Println("COMPATIBLE")
	return nil
}

func runDiff(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: diff <subject> <file>")
	}
	subject, file := args[0], args[1]
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	latest, err := reg.GetLatest(subject)
	if err != nil {
		return err
	}
	changes, err := compat.Diff(latest.Content, string(content))
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		fmt.Println("no structural changes")
		return nil
	}
	for _, c := range changes {
		fmt.Println(c)
	}
	return nil
}

// runHTTPRegister is a helper that registers via HTTP (for external servers).
func runHTTPRegister(serverAddr, subject, content string, format schema.Format) (int, error) {
	body, _ := json.Marshal(map[string]string{"schema": content, "schemaType": string(format)})
	resp, err := http.Post(
		fmt.Sprintf("http://%s/subjects/%s/versions", serverAddr, subject),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	return result["id"], nil
}

func detectFormat(filename string) schema.Format {
	switch {
	case hasSuffix(filename, ".proto"):
		return schema.FormatProtobuf
	case hasSuffix(filename, ".avsc", ".avro"):
		return schema.FormatAvro
	default:
		return schema.FormatJSONSchema
	}
}

func hasSuffix(s string, suffixes ...string) bool {
	for _, suf := range suffixes {
		if len(s) >= len(suf) && s[len(s)-len(suf):] == suf {
			return true
		}
	}
	return false
}
