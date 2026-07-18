package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/PrashikshitSaini/Deja/internal/config"
)

func TestCLIImportQueryAndPick(t *testing.T) {
	directory := t.TempDir()
	historyPath := filepath.Join(directory, ".zsh_history")
	storePath := filepath.Join(directory, "history.jsonl")
	resultsPath := filepath.Join(directory, "results.json")
	history := strings.Join([]string{
		": 1700000000:1;git status",
		": 1700000001:1;git status --short",
		": 1700000002:1;git log --oneline main",
		"",
	}, "\n")
	if err := os.WriteFile(historyPath, []byte(history), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"import", "--history-file", historyPath, "--store", storePath},
		strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import code = %d, stderr = %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "3" {
		t.Fatalf("import output = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"query", "--query-stdin", "--store", storePath, "--cwd", directory,
		"--results-file", resultsPath, "--format", "plain", "--color", "never",
	}, strings.NewReader("git st"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("query code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "git status") || strings.Contains(stdout.String(), "git log") {
		t.Fatalf("query output = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"pick", "--results-file", resultsPath, "--index", "1"},
		strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pick code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "git status") || strings.HasSuffix(stdout.String(), "\n") {
		t.Fatalf("pick output = %q", stdout.String())
	}
}

func TestZshIntegrationNeverBindsEnterOrExecutesASelection(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "shell", "deja.zsh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	enterBinding := regexp.MustCompile(`(?m)bindkey[^\n]*(\^M|\\r|accept-line)`)
	executionCall := regexp.MustCompile(`(?m)^\s*zle\s+\.?accept-line`)
	if enterBinding.MatchString(text) {
		t.Fatal("Deja must not bind Enter")
	}
	if executionCall.MatchString(text) {
		t.Fatal("Deja must not invoke accept-line")
	}
	if !strings.Contains(text, `BUFFER="${selection}"`) {
		t.Fatal("selection must be assigned to the editable Zsh buffer")
	}
}

func TestZshHistoryFallbackSuppressesPaletteAndBindsActiveKeymap(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "shell", "deja.zsh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	upFallback := regexp.MustCompile(`(?s)zle \.up-line-or-history\s+_deja_suppressed=1\s+_deja_last_buffer="\$\{BUFFER\}"`)
	downFallback := regexp.MustCompile(`(?s)zle \.down-line-or-history\s+_deja_suppressed=1\s+_deja_last_buffer="\$\{BUFFER\}"`)
	if !upFallback.MatchString(text) || !downFallback.MatchString(text) {
		t.Fatal("normal Zsh history navigation must suppress the Deja palette")
	}
	if !strings.Contains(text, "for _deja_keymap in main emacs viins") {
		t.Fatal("Deja must bind the active main keymap")
	}
	if !strings.Contains(text, "terminfo[kcuu1]") || !strings.Contains(text, "terminfo[kcud1]") {
		t.Fatal("Deja must bind terminal-provided arrow sequences")
	}
}

func TestZshPalettePassesTerminalWidthToNativeRenderer(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "shell", "deja.zsh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, `--width "${row_width}"`) {
		t.Fatal("Zsh integration must pass the terminal width to Deja")
	}
	if !strings.Contains(text, "--color auto") {
		t.Fatal("Zsh integration must allow configured palette colors")
	}
	if !strings.Contains(text, "--zle-meta") || !strings.Contains(text, `--visible-rows "${DEJA_LIMIT}"`) {
		t.Fatal("Zsh integration must separate the scrolling viewport from the candidate pool")
	}
	if !strings.Contains(text, "commands[deja]") {
		t.Fatal("packaged shell integration must discover a deja binary installed on PATH")
	}
}

func TestZshIntegrationCarriesForwardLegacyEnvironment(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "shell", "deja.zsh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, name := range []string{"BIN", "CONFIG", "STORE", "LIMIT"} {
		legacy := "DEZA_" + name
		current := "DEJA_" + name
		if !strings.Contains(text, legacy) || !strings.Contains(text, current) {
			t.Fatalf("missing %s to %s migration bridge", legacy, current)
		}
	}
}

func TestZshPaletteWindowFollowsSelectionBeyondVisibleRows(t *testing.T) {
	plugin, err := filepath.Abs(filepath.Join("..", "..", "shell", "deja.zsh"))
	if err != nil {
		t.Fatal(err)
	}
	script := strings.Join([]string{
		"typeset -gx DEJA_BIN=/usr/bin/true",
		"source " + plugin,
		`_deja_candidate_lines=(one two three four five six seven eight nine ten)`,
		`_deja_visible_rows=6`,
		`_deja_selected=7`,
		`_deja_update_window`,
		`print -r -- "${_deja_window_start}:${_deja_window_end}"`,
		`_deja_selected=10`,
		`_deja_update_window`,
		`print -r -- "${_deja_window_start}:${_deja_window_end}"`,
		`_deja_selected=1`,
		`_deja_update_window`,
		`print -r -- "${_deja_window_start}:${_deja_window_end}"`,
	}, "\n")
	output, err := exec.Command("zsh", "-dfi", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("Zsh window check failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "2:7\n5:10\n1:6" {
		t.Fatalf("window positions = %q", output)
	}
}

func TestCLIVisibleLimitDoesNotTruncateCandidatePool(t *testing.T) {
	directory := t.TempDir()
	historyPath := filepath.Join(directory, ".zsh_history")
	storePath := filepath.Join(directory, "history.jsonl")
	resultsPath := filepath.Join(directory, "results.json")
	entries := make([]string, 0, 11)
	for index := 1; index <= 10; index++ {
		entries = append(entries, fmt.Sprintf(": %d:1;git checkout branch-%02d", 1700000000+index, index))
	}
	entries = append(entries, "")
	if err := os.WriteFile(historyPath, []byte(strings.Join(entries, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{
		"import", "--history-file", historyPath, "--store", storePath,
	}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("import failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := run([]string{
		"query", "--store", storePath, "--results-file", resultsPath,
		"--format", "zle", "--zle-meta", "git", "checkout",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("query code = %d, stderr = %q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 11 || lines[0] != "__DEJA_META__\t6" {
		t.Fatalf("query returned %d lines: %#v", len(lines), lines)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{
		"pick", "--results-file", resultsPath, "--index", "10",
	}, strings.NewReader(""), &stdout, &stderr); code != 0 || !strings.HasPrefix(stdout.String(), "git checkout branch-") {
		t.Fatalf("pick tenth candidate code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestCLIConfigRedactsAndAggregatesCommitMessages(t *testing.T) {
	directory := t.TempDir()
	historyPath := filepath.Join(directory, ".zsh_history")
	storePath := filepath.Join(directory, "history.jsonl")
	resultsPath := filepath.Join(directory, "results.json")
	history := strings.Join([]string{
		`: 1700000000:1;git commit -m "first private message"`,
		`: 1700000001:1;git commit -m "second private message"`,
		"",
	}, "\n")
	if err := os.WriteFile(historyPath, []byte(history), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"import", "--history-file", historyPath, "--store", storePath}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("import code = %d, stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := run([]string{
		"query", "--store", storePath, "--results-file", resultsPath,
		"--format", "plain", "git", "commit",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("query code = %d, stderr = %q", code, stderr.String())
	}
	output := stdout.String()
	if strings.Contains(output, "private message") || !strings.Contains(output, "<message>") || !strings.Contains(output, "×2") {
		t.Fatalf("redacted aggregate output = %q", output)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"pick", "--results-file", resultsPath, "--index", "1"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pick code = %d, stderr = %q", code, stderr.String())
	}
	if stdout.String() != `git commit -m ""` {
		t.Fatalf("redacted selection = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"query", "--store", storePath, "--format", "zle", "--color", "auto", "--zle-meta", "git", "commit",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ZLE query code = %d, stderr = %q", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "\x1b") || !strings.HasPrefix(stdout.String(), "__DEJA_META__\t6\n🔴") {
		t.Fatalf("ZLE output is not terminal-safe themed text: %q", stdout.String())
	}
}

func TestCLIHonorsConfiguredMinimumQueryLength(t *testing.T) {
	directory := t.TempDir()
	storePath := filepath.Join(directory, "history.jsonl")
	configPath := filepath.Join(directory, "config.json")
	settings := config.Defaults()
	settings.Display.MinimumQueryLength = 4
	if err := config.Write(configPath, settings, false); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"record", "--store", storePath, "--timestamp", "1700000000", "git", "status",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("record code = %d, stderr = %q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{
		"query", "--store", storePath, "--config", configPath, "--format", "plain", "git",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stdout.Len() != 0 {
		t.Fatalf("short query code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}
