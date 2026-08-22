package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/PrashikshitSaini/Deja/internal/config"
	"github.com/PrashikshitSaini/Deja/internal/matcher"
)

func runConfig(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		fmt.Fprintln(stderr, "Usage: deja config <init|show|check|path|explain> [options]")
		return 2
	}
	switch arguments[0] {
	case "path":
		fmt.Fprintln(stdout, config.DefaultPath())
		return 0

	case "init":
		set := flagSet("deja config init", stderr)
		path := set.String("file", "", "configuration file")
		force := set.Bool("force", false, "replace an existing configuration")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		if *path == "" {
			*path = config.DefaultPath()
		}
		if err := config.Write(*path, config.Defaults(), *force); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintln(stdout, *path)
		return 0

	case "show", "check":
		set := flagSet("deja config "+arguments[0], stderr)
		path := set.String("file", "", "configuration file")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		settings, loadedPath, err := config.Load(*path)
		if err != nil {
			return fail(stderr, err)
		}
		if arguments[0] == "check" {
			fmt.Fprintf(stdout, "ok: %s\n", loadedPath)
			return 0
		}
		fmt.Fprintf(stdout, "config: %s\n", loadedPath)
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(settings); err != nil {
			return fail(stderr, err)
		}
		return 0

	case "explain":
		set := flagSet("deja config explain", stderr)
		path := set.String("file", "", "configuration file")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		command := strings.Join(set.Args(), " ")
		family := matcher.Family(command)
		if family == "" {
			fmt.Fprintln(stderr, "deja: a command is required")
			return 2
		}
		settings, loadedPath, err := config.Load(*path)
		if err != nil {
			return fail(stderr, err)
		}
		insert, display, visible := settings.Prepare(command, family)
		fmt.Fprintf(stdout, "config: %s\nvisible: %t\n", loadedPath, visible)
		if visible {
			if display == "" {
				display = insert
			}
			fmt.Fprintf(stdout, "display: %s\ninsert: %s\n", display, insert)
		}
		return 0

	default:
		fmt.Fprintf(stderr, "deja: unknown config command %q\n", arguments[0])
		return 2
	}
}
