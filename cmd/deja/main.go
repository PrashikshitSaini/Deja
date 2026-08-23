package main

import (
	"fmt"
	"io"
	"os"
)

const version = "0.3.1"

func usage(writer io.Writer) {
	fmt.Fprintln(writer, "Deja — local command-history variants for Zsh")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Usage: deja <command> [options]")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Commands:")
	fmt.Fprintln(writer, "  import   Import an existing Zsh history file")
	fmt.Fprintln(writer, "  record   Record one completed command")
	fmt.Fprintln(writer, "  query    Find distinct matching command variants")
	fmt.Fprintln(writer, "  pick     Return a cached selection without executing it")
	fmt.Fprintln(writer, "  config   Initialize, inspect, validate, or explain settings")
	fmt.Fprintln(writer, "  stats    Show local index statistics")
	fmt.Fprintln(writer, "  doctor   Check the local Deja environment")
	fmt.Fprintln(writer, "  version  Print the Deja version")
}

func run(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		usage(stderr)
		return 2
	}

	switch arguments[0] {
	case "version", "--version", "-version":
		fmt.Fprintf(stdout, "deja %s\n", version)
		return 0

	case "config":
		return runConfig(arguments[1:], stdout, stderr)

	case "import":
		return runImport(arguments[1:], stdout, stderr)

	case "record":
		return runRecord(arguments[1:], stdin, stderr)

	case "query":
		return runQuery(arguments[1:], stdin, stdout, stderr)

	case "pick":
		return runPick(arguments[1:], stdout, stderr)

	case "stats":
		return runStats(arguments[1:], stdout, stderr)

	case "doctor":
		return runDoctor(stdout)

	default:
		fmt.Fprintf(stderr, "deja: unknown command %q\n\n", arguments[0])
		usage(stderr)
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
