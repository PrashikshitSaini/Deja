package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PrashikshitSaini/Deja/internal/config"
	"github.com/PrashikshitSaini/Deja/internal/history"
	"github.com/PrashikshitSaini/Deja/internal/matcher"
	"github.com/PrashikshitSaini/Deja/internal/render"
	"github.com/PrashikshitSaini/Deja/internal/store"
)

const version = "0.3.0"

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

func flagSet(name string, stderr io.Writer) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(stderr)
	return set
}

func readAll(reader io.Reader) (string, error) {
	content, err := io.ReadAll(reader)
	return string(content), err
}

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
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
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
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		if arguments[0] == "check" {
			fmt.Fprintf(stdout, "ok: %s\n", loadedPath)
			return 0
		}
		fmt.Fprintf(stdout, "config: %s\n", loadedPath)
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(settings); err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
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
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
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
		set := flagSet("deja import", stderr)
		historyFile := set.String("history-file", "", "Zsh history file")
		storePath := set.String("store", "", "Deja event store")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		if *historyFile == "" {
			fmt.Fprintln(stderr, "deja: --history-file is required")
			return 2
		}
		absolute, err := filepath.Abs(*historyFile)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		entries, err := history.ReadZsh(absolute)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		imported, err := store.New(*storePath).Import(entries, absolute)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, imported)
		return 0

	case "record":
		set := flagSet("deja record", stderr)
		fromStdin := set.Bool("stdin", false, "read the command from stdin")
		cwd := set.String("cwd", "", "working directory")
		exitStatus := set.String("exit-status", "", "command exit status")
		timestamp := set.Float64("timestamp", 0, "execution timestamp")
		duration := set.Float64("duration", 0, "execution duration")
		storePath := set.String("store", "", "Deja event store")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		var command string
		var err error
		if *fromStdin {
			command, err = readAll(stdin)
		} else {
			command = strings.Join(set.Args(), " ")
		}
		if err != nil || strings.TrimSpace(command) == "" {
			fmt.Fprintln(stderr, "deja: a command is required")
			return 2
		}
		var status *int
		if *exitStatus != "" {
			value, parseErr := strconv.Atoi(*exitStatus)
			if parseErr != nil {
				fmt.Fprintf(stderr, "deja: invalid exit status: %v\n", parseErr)
				return 2
			}
			status = &value
		}
		recorded, err := store.New(*storePath).Record(command, *cwd, status, *timestamp, *duration)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		if !recorded {
			return 1
		}
		return 0

	case "query":
		set := flagSet("deja query", stderr)
		fromStdin := set.Bool("query-stdin", false, "read the query from stdin")
		cwd, _ := os.Getwd()
		currentDirectory := set.String("cwd", cwd, "current working directory")
		limit := set.Int("limit", 0, "override the configured candidate pool limit")
		visibleRows := set.Int("visible-rows", 0, "override the configured visible row count")
		width := set.Int("width", 0, "maximum visible row width")
		zleMeta := set.Bool("zle-meta", false, "include Zsh palette metadata")
		resultsFile := set.String("results-file", "", "write raw candidates for selection")
		format := set.String("format", "plain", "plain, zle, or json")
		color := set.String("color", "auto", "auto, always, or never")
		storePath := set.String("store", "", "Deja event store")
		configPath := set.String("config", "", "Deja configuration file")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		if *zleMeta && *format != "zle" {
			fmt.Fprintln(stderr, "deja: --zle-meta requires --format zle")
			return 2
		}
		if *visibleRows < 0 || *visibleRows > 50 {
			fmt.Fprintln(stderr, "deja: --visible-rows must be between 1 and 50")
			return 2
		}
		if *limit < 0 || *limit > 5000 {
			fmt.Fprintln(stderr, "deja: --limit must be between 1 and 5000")
			return 2
		}
		var query string
		var err error
		if *fromStdin {
			query, err = readAll(stdin)
		} else {
			query = strings.Join(set.Args(), " ")
		}
		if err != nil || strings.TrimSpace(query) == "" {
			return 0
		}
		settings, _, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		if utf8.RuneCountInString(strings.TrimSpace(query)) < settings.Display.MinimumQueryLength {
			return 0
		}
		queryLimit := settings.Display.CandidateLimit
		if *limit > 0 {
			queryLimit = *limit
		}
		paletteRows := settings.Display.Limit
		if *visibleRows > 0 {
			paletteRows = *visibleRows
		}
		candidates, err := store.New(*storePath).SearchWithOptions(query, *currentDirectory, store.SearchOptions{
			Limit:       queryLimit,
			MinimumUses: settings.Display.MinimumUses,
			MaxAgeDays:  settings.Display.MaximumAgeDays,
			Prepare:     settings.Prepare,
		})
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		if *resultsFile != "" {
			if err := render.WriteResults(*resultsFile, candidates); err != nil {
				fmt.Fprintf(stderr, "deja: %v\n", err)
				return 2
			}
		}
		if *format == "json" {
			if err := json.NewEncoder(stdout).Encode(candidates); err != nil {
				fmt.Fprintf(stderr, "deja: %v\n", err)
				return 2
			}
			return 0
		}
		if *format != "plain" && *format != "zle" {
			fmt.Fprintf(stderr, "deja: unsupported format %q\n", *format)
			return 2
		}
		if *zleMeta {
			fmt.Fprintf(stdout, "__DEJA_META__\t%d\n", paletteRows)
		}
		if *color != "auto" && *color != "always" && *color != "never" {
			fmt.Fprintf(stderr, "deja: unsupported color mode %q\n", *color)
			return 2
		}
		rankSymbols := *format == "zle" && *color != "never" && settings.Display.Colors.Enabled
		colored := *color == "always" && *format != "zle"
		options := render.Options{
			Color:                colored,
			RankSymbols:          rankSymbols,
			Width:                *width,
			RankColors:           settings.Display.Colors.Rank,
			FamilyColor:          settings.Display.Colors.Family,
			DifferenceColor:      settings.Display.Colors.Difference,
			MetadataColor:        settings.Display.Colors.Metadata,
			ShowUsage:            settings.Display.Metadata.Usage,
			ShowSuccessRate:      settings.Display.Metadata.SuccessRate,
			ShowCurrentDirectory: settings.Display.Metadata.CurrentDirectory,
		}
		for _, row := range render.Rows(candidates, options) {
			fmt.Fprintln(stdout, row)
		}
		return 0

	case "pick":
		set := flagSet("deja pick", stderr)
		resultsFile := set.String("results-file", "", "cached result file")
		index := set.Int("index", 0, "one-based selection index")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		selection, err := render.ReadResult(*resultsFile, *index)
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		fmt.Fprint(stdout, selection)
		return 0

	case "stats":
		set := flagSet("deja stats", stderr)
		storePath := set.String("store", "", "Deja event store")
		if err := set.Parse(arguments[1:]); err != nil {
			return 2
		}
		historyStore := store.New(*storePath)
		count, err := historyStore.Count()
		if err != nil {
			fmt.Fprintf(stderr, "deja: %v\n", err)
			return 2
		}
		fmt.Fprintf(stdout, "store: %s\nevents: %d\n", historyStore.Path, count)
		return 0

	case "doctor":
		zsh, _ := exec.LookPath("zsh")
		if zsh == "" {
			zsh = "not found"
		}
		_, configPath, configErr := config.Load("")
		configStatus := "ok"
		if configErr != nil {
			configStatus = configErr.Error()
		}
		fmt.Fprintf(stdout, "deja: %s\ngo: %s\nzsh: %s\nstore: %s\nconfig: %s\nconfig_status: %s\nlocal_only: yes\nchecked_at: %d\n",
			version, runtime.Version(), zsh, store.DefaultPath(), configPath, configStatus, time.Now().Unix())
		if zsh == "not found" {
			return 1
		}
		if configErr != nil {
			return 1
		}
		return 0

	default:
		fmt.Fprintf(stderr, "deja: unknown command %q\n\n", arguments[0])
		usage(stderr)
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
