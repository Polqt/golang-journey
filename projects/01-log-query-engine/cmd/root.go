package cmd

import (
	"errors"
	"fmt"
	"os"
)

// Run is the top-level dispatcher for the logql CLI.
func Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "query", "q":
		return runQuery(args[1:])
	case "tail":
		return runTail(args[1:])
	case "repl":
		return runREPL(args[1:])
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q — run 'logql help'", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `logql — SQL queries over structured log files

USAGE:
  logql query [flags] <sql>    Execute a SQL query against log files
  logql tail  [flags]          Live-tail logs with optional WHERE filter
  logql repl  [file...]        Interactive query REPL

FLAGS (query):
  -f, --file <path>    Log file(s) to query (can repeat; use _ as table name)
  -o, --format         Output format: table (default), json, csv
      --explain        Show query execution plan

FLAGS (tail):
  -f, --file <path>    File to tail
  -q, --query <sql>    WHERE clause to filter

EXAMPLES:
  logql query -f nginx.ndjson "SELECT status, COUNT(*) FROM _ GROUP BY status"
  logql tail -f /var/log/app.logfmt -q "WHERE level = 'error'"
  logql repl app.logfmt`)
}

func runQuery(args []string) error {
	// TODO: parse -f/--file, -o/--format, --explain flags
	// TODO: parse the SQL argument
	// TODO: build engine.Engine, call Execute(sql, files)
	// TODO: render results
	return errors.New("query: not yet implemented")
}

func runTail(args []string) error {
	// TODO: parse flags
	// TODO: open file, seek to end
	// TODO: watch for new lines (poll or inotify via syscall)
	// TODO: apply WHERE filter to each new line
	// TODO: print matching lines
	return errors.New("tail: not yet implemented")
}

func runREPL(args []string) error {
	// TODO: load files from args
	// TODO: print welcome banner
	// TODO: read-eval-print loop using bufio.Scanner on os.Stdin
	// TODO: handle meta-commands: \timing, \format, \explain, \quit
	return errors.New("repl: not yet implemented")
}
