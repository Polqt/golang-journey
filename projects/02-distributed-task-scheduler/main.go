package main

import (
	"fmt"
	"os"

	"github.com/Polqt/scheduler/scheduler/store"
	"github.com/Polqt/scheduler/scheduler/worker"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}
	switch os.Args[1] {
	case "server":
		runServer()
	case "submit":
		runSubmit(os.Args[2:])
	case "status":
		runStatus()
	case "dlq":
		runDLQ(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runServer() {
	// TODO:
	// 1. Initialize SQLite store
	// 2. Start worker pool (Concurrency: 10)
	// 3. Start scheduler loop (check for recurring tasks every pollInterval)
	// 4. Start HTTP API server
	// 5. Handle SIGTERM → graceful shutdown
	fmt.Println("starting scheduler server...")
	_ = store.Open
	_ = worker.NewPool
}

func runSubmit(args []string) {
	// TODO: parse flags: --type, --payload, --cron, --max-attempts
	// POST to scheduler API or write directly to SQLite
	fmt.Println("submit: not yet implemented")
}

func runStatus() {
	// TODO: GET /stats and pretty-print
	fmt.Println("status: not yet implemented")
}

func runDLQ(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: scheduler dlq [list|retry <id>]")
		return
	}
	switch args[0] {
	case "list":
		fmt.Println("dlq list: not yet implemented")
	case "retry":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: scheduler dlq retry <task-id>")
			return
		}
		fmt.Printf("dlq retry %s: not yet implemented\n", args[1])
	}
}

func printUsage() {
	fmt.Print(`scheduler — distributed task scheduler

USAGE:
  scheduler server                    Start scheduler + worker server
  scheduler submit [flags]            Submit a task
    --type string        Task type name
    --payload string     JSON payload
    --cron string        Cron expression (recurring tasks)
    --max-attempts int   Max retry attempts (default 3)
  scheduler status                    Show queue stats
  scheduler dlq list                  List dead-letter queue
  scheduler dlq retry <id>            Retry a dead task
`)
}
