package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/PrashikshitSaini/Deja/internal/history"
	"github.com/PrashikshitSaini/Deja/internal/store"
)

func runPurge(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	set := flagSet("deja purge", stderr)
	exact := set.String("exact", "", "match one complete command")
	contains := set.String("contains", "", "match commands containing literal text")
	fromStdin := set.Bool("stdin", false, "read an exact command from stdin")
	ignoreCase := set.Bool("ignore-case", false, "match without case sensitivity")
	storePath := set.String("store", "", "Deja event store")
	historyFile := set.String("history-file", "", "also purge this Zsh history file")
	force := set.Bool("force", false, "apply the purge; otherwise only count matches")
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	selectors := 0
	if *exact != "" {
		selectors++
	}
	if *contains != "" {
		selectors++
	}
	if *fromStdin {
		selectors++
	}
	if selectors != 1 {
		fmt.Fprintln(stderr, "deja: choose exactly one of --exact, --contains, or --stdin")
		return 2
	}
	if *fromStdin {
		value, err := readAll(stdin)
		if err != nil {
			return fail(stderr, err)
		}
		*exact = strings.TrimSpace(value)
		if *exact == "" {
			fmt.Fprintln(stderr, "deja: stdin did not contain a command")
			return 2
		}
	}
	needle := *exact
	matchExact := true
	if *contains != "" {
		needle = *contains
		matchExact = false
	}
	if *ignoreCase {
		needle = strings.ToLower(needle)
	}
	match := func(command string) bool {
		candidate := command
		if *ignoreCase {
			candidate = strings.ToLower(candidate)
		}
		if matchExact {
			return candidate == needle
		}
		return strings.Contains(candidate, needle)
	}

	historyStore := store.New(*storePath)
	storeMatches, err := historyStore.PurgeCommands(match, false)
	if err != nil {
		return fail(stderr, err)
	}
	historyMatches := 0
	if *historyFile != "" {
		historyMatches, err = history.PurgeZsh(*historyFile, match, false)
		if err != nil {
			return fail(stderr, err)
		}
	}
	if !*force {
		fmt.Fprintf(stdout, "dry_run: true\nstore: %s\nstore_matches: %d\n", historyStore.Path, storeMatches)
		if *historyFile != "" {
			fmt.Fprintf(stdout, "history_file: %s\nhistory_matches: %d\n", *historyFile, historyMatches)
			fmt.Fprintln(stdout, "Close other active Zsh sessions before using --force.")
		}
		fmt.Fprintln(stdout, "Run again with --force to permanently delete these records.")
		return 0
	}

	historyRemoved := 0
	if *historyFile != "" {
		fmt.Fprintln(stderr, "deja: ensure other active Zsh sessions are closed before rewriting history")
		historyRemoved, err = history.PurgeZsh(*historyFile, match, true)
		if err != nil {
			return fail(stderr, err)
		}
	}
	storeRemoved, err := historyStore.PurgeCommands(match, true)
	if err != nil {
		if *historyFile != "" {
			fmt.Fprintf(stderr, "deja: Zsh history was purged, but the store purge failed: %v\n", err)
			return 1
		}
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "store: %s\nstore_removed: %d\n", historyStore.Path, storeRemoved)
	if *historyFile != "" {
		fmt.Fprintf(stdout, "history_file: %s\nhistory_removed: %d\n", *historyFile, historyRemoved)
		fmt.Fprintln(stdout, "Close other active Zsh sessions so they cannot rewrite purged in-memory history.")
	}
	return 0
}
