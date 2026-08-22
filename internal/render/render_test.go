package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/PrashikshitSaini/Deja/internal/model"
)

func TestRowsHighlightDifferingFlagsAndArguments(t *testing.T) {
	candidates := []model.Candidate{
		{Command: "git log --oneline main", Uses: 2},
		{Command: "git log --graph develop", Uses: 1},
	}
	rows := Rows(candidates, DefaultOptions(false))
	if len(rows) != 2 {
		t.Fatalf("Rows() returned %d rows", len(rows))
	}
	if !strings.Contains(rows[0], "[--oneline]") || !strings.Contains(rows[0], "[main]") {
		t.Fatalf("first row did not highlight differences: %q", rows[0])
	}
	if strings.Contains(rows[0], "[git]") || strings.Contains(rows[0], "[log]") {
		t.Fatalf("common tokens were highlighted: %q", rows[0])
	}
}

func TestRowsShortenLongCommandsAtWordBoundaries(t *testing.T) {
	candidates := []model.Candidate{{
		Command: "git commit -m the beginning words continue through a very long command and finish with the final words",
		Uses:    4,
	}}
	options := DefaultOptions(false)
	options.Width = 64
	rows := Rows(candidates, options)
	if utf8.RuneCountInString(rows[0]) > options.Width {
		t.Fatalf("row width = %d, want <= %d: %q", utf8.RuneCountInString(rows[0]), options.Width, rows[0])
	}
	if !strings.HasPrefix(rows[0], "git [commit] [-m]") {
		t.Fatalf("shortened row lost its beginning: %q", rows[0])
	}
	if !strings.Contains(rows[0], " ... ") {
		t.Fatalf("shortened row has no middle omission: %q", rows[0])
	}
	if !strings.HasSuffix(rows[0], "[final] [words]  ×4") {
		t.Fatalf("shortened row lost its ending: %q", rows[0])
	}
}

func TestRowsKeepUsefulTailOfOneVeryLongFinalArgument(t *testing.T) {
	candidates := []model.Candidate{{
		Command: "git commit -- /Volumes/CORSAIR/Projects/a-very-long-project-name/web/app/office/team/page.tsx",
		Uses:    2,
	}}
	options := DefaultOptions(false)
	options.Width = 48
	row := Rows(candidates, options)[0]
	if utf8.RuneCountInString(row) > options.Width {
		t.Fatalf("row width = %d, want <= %d: %q", utf8.RuneCountInString(row), options.Width, row)
	}
	if !strings.HasSuffix(row, "team/page.tsx]  ×2") {
		t.Fatalf("shortened row lost the useful path tail: %q", row)
	}
}

func TestRowsApplyColorsByRank(t *testing.T) {
	candidates := []model.Candidate{
		{Command: "git status", Uses: 4},
		{Command: "git push", Uses: 2},
	}
	options := DefaultOptions(true)
	options.RankColors = []string{"dark-red", "orange"}
	rows := Rows(candidates, options)
	if !strings.Contains(rows[0], "\x1b[38;5;88m") {
		t.Fatalf("top row is not dark red: %q", rows[0])
	}
	if !strings.Contains(rows[1], "\x1b[38;5;166m") {
		t.Fatalf("second row is not orange: %q", rows[1])
	}
}

func TestRowsUseTerminalSafeRankNumbersForZLE(t *testing.T) {
	candidates := []model.Candidate{
		{Command: "git status", Uses: 4},
		{Command: "git push", Uses: 2},
	}
	options := DefaultOptions(false)
	options.RankSymbols = true
	rows := Rows(candidates, options)
	if !strings.HasPrefix(rows[0], " 1 ") || !strings.HasPrefix(rows[1], " 2 ") {
		t.Fatalf("rank numbers = %#v", rows)
	}
	if strings.Contains(rows[0], "\x1b") || strings.Contains(rows[1], "\x1b") {
		t.Fatalf("ZLE-safe rows contain ANSI controls: %#v", rows)
	}
}

func TestRowsAllowMetadataToBeDisabled(t *testing.T) {
	candidate := model.Candidate{
		Command: "git status", Uses: 9, Successes: 8, StatusCount: 9, CWDHits: 3,
	}
	options := DefaultOptions(false)
	options.ShowUsage = false
	options.ShowSuccessRate = false
	options.ShowCurrentDirectory = false
	row := Rows([]model.Candidate{candidate}, options)[0]
	if row != "git [status]" {
		t.Fatalf("row with disabled metadata = %q", row)
	}
}

func TestReadResultReturnsTextWithoutExecutingIt(t *testing.T) {
	directory := t.TempDir()
	results := filepath.Join(directory, "results.json")
	marker := filepath.Join(directory, "must-not-exist")
	command := "touch " + marker
	if err := WriteResults(results, []model.Candidate{{Command: command, Family: "touch"}}); err != nil {
		t.Fatal(err)
	}
	selected, err := ReadResult(results, 1)
	if err != nil {
		t.Fatal(err)
	}
	if selected != command {
		t.Fatalf("ReadResult() = %q", selected)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("selection was executed; marker stat error = %v", err)
	}
}
