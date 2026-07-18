package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PrashikshitSaini/Deja/internal/history"
)

func intPointer(value int) *int { return &value }

func TestImportDeduplicatesAndSearchGroupsDistinctVariants(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deja", "history.jsonl")
	historyStore := New(path)
	entries := []history.Entry{
		{Command: "git status", OccurredAt: 100, Ordinal: 1},
		{Command: "git status", OccurredAt: 200, Ordinal: 2},
		{Command: "git status --short", OccurredAt: 300, Ordinal: 3},
		{Command: "docker ps", OccurredAt: 400, Ordinal: 4},
	}
	added, err := historyStore.Import(entries, "/tmp/.zsh_history")
	if err != nil {
		t.Fatal(err)
	}
	if added != 4 {
		t.Fatalf("first import added %d entries", added)
	}
	added, err = historyStore.Import(entries, "/tmp/.zsh_history")
	if err != nil {
		t.Fatal(err)
	}
	if added != 0 {
		t.Fatalf("second import added %d entries, want 0", added)
	}

	recorded, err := historyStore.Record("git status --short", "/repo", intPointer(0), 500, 0.1)
	if err != nil || !recorded {
		t.Fatalf("Record() = %v, %v", recorded, err)
	}
	candidates, err := historyStore.Search("git st", "/repo", 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("Search() returned %d candidates", len(candidates))
	}
	if candidates[0].Command != "git status --short" || candidates[0].Uses != 2 || candidates[0].CWDHits != 1 {
		t.Fatalf("top candidate = %#v", candidates[0])
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("store permissions = %o", info.Mode().Perm())
	}
}

func BenchmarkSearchTwentyThousandEvents(b *testing.B) {
	path := filepath.Join(b.TempDir(), "history.jsonl")
	historyStore := New(path)
	entries := make([]history.Entry, 0, 20_000)
	for index := 0; index < 20_000; index++ {
		entries = append(entries, history.Entry{
			Command:    fmt.Sprintf("git checkout branch-%d", index%250),
			OccurredAt: float64(1_700_000_000 + index),
			Ordinal:    index + 1,
		})
	}
	if _, err := historyStore.Import(entries, "/synthetic/.zsh_history"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := historyStore.Search("git checkout", "/repo", 6); err != nil {
			b.Fatal(err)
		}
	}
}

func TestLiveRecordMatchesExtendedHistoryIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	historyStore := New(path)
	if _, err := historyStore.Record("aws s3 ls", "/repo", intPointer(0), 1_700_000_000.25, 0.5); err != nil {
		t.Fatal(err)
	}
	added, err := historyStore.Import([]history.Entry{{
		Command: "aws s3 ls", OccurredAt: 1_700_000_000, Ordinal: 1,
	}}, "/tmp/history")
	if err != nil {
		t.Fatal(err)
	}
	if added != 0 {
		t.Fatalf("import added %d duplicate live events", added)
	}
}

func TestStoreDoesNotChangeExistingParentPermissions(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	historyStore := New(filepath.Join(parent, "history.jsonl"))
	if _, err := historyStore.Record("ls -la", parent, intPointer(0), 500, 0.1); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(parent)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("existing parent permissions changed to %o", info.Mode().Perm())
	}
}

func TestDefaultPathPrefersDejaStoreAndSupportsLegacyName(t *testing.T) {
	t.Setenv("DEJA_STORE", "/new/history.jsonl")
	t.Setenv("DEZA_STORE", "/legacy/history.jsonl")
	if got := DefaultPath(); got != "/new/history.jsonl" {
		t.Fatalf("DefaultPath() = %q", got)
	}

	t.Setenv("DEJA_STORE", "")
	if got := DefaultPath(); got != "/legacy/history.jsonl" {
		t.Fatalf("legacy DefaultPath() = %q", got)
	}
}

func TestExactFamilyDoesNotLeakPrefixFamilies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	historyStore := New(path)
	entries := []history.Entry{
		{Command: "git status", OccurredAt: 100, Ordinal: 1},
		{Command: "git-lfs status", OccurredAt: 200, Ordinal: 2},
	}
	if _, err := historyStore.Import(entries, "/tmp/history"); err != nil {
		t.Fatal(err)
	}
	candidates, err := historyStore.Search("git", "", 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Family != "git" {
		t.Fatalf("exact family search returned %#v", candidates)
	}
}

func TestSearchOptionsFilterOneOffAndStaleCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	historyStore := New(path)
	now := float64(time.Now().Unix())
	entries := []history.Entry{
		{Command: "git status", OccurredAt: now - 60, Ordinal: 1},
		{Command: "git status", OccurredAt: now - 30, Ordinal: 2},
		{Command: "git push", OccurredAt: now - 20, Ordinal: 3},
		{Command: "git checkout ancient", OccurredAt: now - 90*86400, Ordinal: 4},
	}
	if _, err := historyStore.Import(entries, "/tmp/history"); err != nil {
		t.Fatal(err)
	}
	candidates, err := historyStore.SearchWithOptions("git", "", SearchOptions{
		Limit:       6,
		MinimumUses: 2,
		MaxAgeDays:  30,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Command != "git status" || candidates[0].Uses != 2 {
		t.Fatalf("filtered candidates = %#v", candidates)
	}
}
