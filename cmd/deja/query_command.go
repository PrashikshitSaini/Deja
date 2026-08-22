package main

import (
	"encoding/json"
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
	"github.com/PrashikshitSaini/Deja/internal/render"
	"github.com/PrashikshitSaini/Deja/internal/store"
)

func runImport(arguments []string, stdout, stderr io.Writer) int {
	set := flagSet("deja import", stderr)
	historyFile := set.String("history-file", "", "Zsh history file")
	storePath := set.String("store", "", "Deja event store")
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	if *historyFile == "" {
		fmt.Fprintln(stderr, "deja: --history-file is required")
		return 2
	}
	absolute, err := filepath.Abs(*historyFile)
	if err != nil {
		return fail(stderr, err)
	}
	entries, err := history.ReadZsh(absolute)
	if err != nil {
		return fail(stderr, err)
	}
	imported, err := store.New(*storePath).Import(entries, absolute)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintln(stdout, imported)
	return 0
}

func runRecord(arguments []string, stdin io.Reader, stderr io.Writer) int {
	set := flagSet("deja record", stderr)
	fromStdin := set.Bool("stdin", false, "read the command from stdin")
	cwd := set.String("cwd", "", "working directory")
	exitStatus := set.String("exit-status", "", "command exit status")
	timestamp := set.Float64("timestamp", 0, "execution timestamp")
	duration := set.Float64("duration", 0, "execution duration")
	storePath := set.String("store", "", "Deja event store")
	if err := set.Parse(arguments); err != nil {
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
		return fail(stderr, err)
	}
	if !recorded {
		return 1
	}
	return 0
}

func runQuery(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
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
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	if *color != "auto" && *color != "always" && *color != "never" {
		fmt.Fprintf(stderr, "deja: unsupported color mode %q\n", *color)
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
	if *format != "plain" && *format != "zle" && *format != "json" {
		fmt.Fprintf(stderr, "deja: unsupported format %q\n", *format)
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
		return fail(stderr, err)
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
		return fail(stderr, err)
	}
	if *resultsFile != "" {
		if err := render.WriteResults(*resultsFile, candidates); err != nil {
			return fail(stderr, err)
		}
	}
	if *format == "json" {
		if err := json.NewEncoder(stdout).Encode(candidates); err != nil {
			return fail(stderr, err)
		}
		return 0
	}
	if *zleMeta {
		fmt.Fprintf(stdout, "__DEJA_META__\t%d\n", paletteRows)
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
}

func runPick(arguments []string, stdout, stderr io.Writer) int {
	set := flagSet("deja pick", stderr)
	resultsFile := set.String("results-file", "", "cached result file")
	index := set.Int("index", 0, "one-based selection index")
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	selection, err := render.ReadResult(*resultsFile, *index)
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprint(stdout, selection)
	return 0
}

func runStats(arguments []string, stdout, stderr io.Writer) int {
	set := flagSet("deja stats", stderr)
	storePath := set.String("store", "", "Deja event store")
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	historyStore := store.New(*storePath)
	count, err := historyStore.Count()
	if err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "store: %s\nevents: %d\n", historyStore.Path, count)
	return 0
}

func runDoctor(stdout io.Writer) int {
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
	if zsh == "not found" || configErr != nil {
		return 1
	}
	return 0
}
