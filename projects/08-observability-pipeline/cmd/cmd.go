// Package cmd wires the observability pipeline CLI.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Polqt/obspipeline/exporter"
	"github.com/Polqt/obspipeline/pipeline"
	"github.com/Polqt/obspipeline/processor"
	"github.com/Polqt/obspipeline/receiver"
)

// Run dispatches subcommands.
func Run(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		printHelp()
		return nil
	}
	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "version":
		fmt.Println("obspipeline v0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp() {
	fmt.Println(`obspipeline — OpenTelemetry-compatible collector pipeline

Commands:
  serve  [otlp-addr] [export-addr]
         Start the pipeline.
         Defaults: receive on :4318, export to stdout.
  version`)
}

func runServe(args []string) error {
	receiveAddr := ":4318"
	exportAddr := ""
	if len(args) > 0 {
		receiveAddr = args[0]
	}
	if len(args) > 1 {
		exportAddr = args[1]
	}

	// Build pipeline.
	p := pipeline.New(1024)

	// Receivers.
	p.AddReceiver(receiver.NewOTLPHTTP(receiveAddr))

	// Processors: tail-sample then batch-flush.
	p.AddProcessor(processor.NewTailSampler(processor.DefaultTailSamplerConfig()))
	p.AddProcessor(processor.NewBatch(500, 5*time.Second))

	// Exporters.
	p.AddExporter(exporter.NewStdout(false))
	if exportAddr != "" {
		p.AddExporter(exporter.NewOTLPHTTP(exportAddr, nil))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := p.Start(ctx); err != nil {
		return fmt.Errorf("pipeline start: %w", err)
	}

	fmt.Printf("obspipeline running — OTLP/HTTP on %s\n", receiveAddr)
	<-ctx.Done()
	fmt.Println("\nshutting down...")
	p.Stop()
	return nil
}
