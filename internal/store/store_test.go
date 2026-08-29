package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestPurgeCommandsDryRunAndAtomicRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	historyStore := New(path)
	entries := []history.Entry{
		{Command: "export OPENAI_API_KEY=private", OccurredAt: 100, Ordinal: 1},
		{Command: "git status", OccurredAt: 200, Ordinal: 2},
		{Command: "OPENAI_API_KEY=private tool", OccurredAt: 300, Ordinal: 3},
	}
	if _, err := historyStore.Import(entries, "/tmp/history"); err != nil {
		t.Fatal(err)
	}
	match := func(command string) bool { return strings.Contains(command, "OPENAI_API_KEY") }
	removed, err := historyStore.PurgeCommands(match, false)
	if err != nil || removed != 2 {
		t.Fatalf("dry run = removed %d, err %v", removed, err)
	}
	if count, err := historyStore.Count(); err != nil || count != 3 {
		t.Fatalf("count after dry run = %d, err %v", count, err)
	}
	removed, err = historyStore.PurgeCommands(match, true)
	if err != nil || removed != 2 {
		t.Fatalf("purge = removed %d, err %v", removed, err)
	}
	if count, err := historyStore.Count(); err != nil || count != 1 {
		t.Fatalf("count after purge = %d, err %v", count, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("purged store permissions = %o", info.Mode().Perm())
	}
}

func TestRecordSkipsPurgeInvocations(t *testing.T) {
	historyStore := New(filepath.Join(t.TempDir(), "history.jsonl"))
	recorded, err := historyStore.Record(
		`deja purge --exact 'export OPENAI_API_KEY=private' --force`, "", intPointer(0), 100, 0.1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if recorded {
		t.Fatal("purge invocation was recorded")
	}
	if count, err := historyStore.Count(); err != nil || count != 0 {
		t.Fatalf("count = %d, err %v", count, err)
	}
}

func TestRecordSkipsPipedPurgeInvocations(t *testing.T) {
	historyStore := New(filepath.Join(t.TempDir(), "history.jsonl"))
	for index, command := range []string{
		`print -rn -- 'export OPENAI_API_KEY=private' | deja purge --stdin --force`,
		`sudo -u root deja purge --exact 'export TOKEN=secret' --force`,
		`env MODE=x deja purge --exact 'export TOKEN=secret' --force`,
		`command -- deja purge --contains TOKEN --force`,
		`if deja purge --contains TOKEN --force; then print purged; fi`,
		`while deja purge --contains TOKEN --force; do break; done`,
		`until deja purge --contains TOKEN --force; do break; done`,
	} {
		recorded, err := historyStore.Record(command, "", intPointer(0), float64(100+index), 0.1)
		if err != nil {
			t.Fatal(err)
		}
		if recorded {
			t.Fatalf("purge invocation %q was recorded", command)
		}
	}
}

func TestRecordKeepsArgumentsThatOnlyMentionDejaPurge(t *testing.T) {
	historyStore := New(filepath.Join(t.TempDir(), "history.jsonl"))
	recorded, err := historyStore.Record(`printf '%s %s\n' deja purge`, "", intPointer(0), 100, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	if !recorded {
		t.Fatal("ordinary command mentioning deja purge was skipped")
	}
}

func TestImportSkipsPurgeInvocations(t *testing.T) {
	historyStore := New(filepath.Join(t.TempDir(), "history.jsonl"))
	imported, err := historyStore.Import([]history.Entry{
		{Command: `deja purge --exact 'export OPENAI_API_KEY=private' --force`, Ordinal: 1},
		{Command: `print -rn -- 'export TOKEN=private' | deja purge --stdin --force`, Ordinal: 2},
		{Command: "git status", Ordinal: 3},
	}, "/tmp/history")
	if err != nil {
		t.Fatal(err)
	}
	if imported != 1 {
		t.Fatalf("imported = %d, want 1", imported)
	}
	if count, err := historyStore.Count(); err != nil || count != 1 {
		t.Fatalf("count = %d, err %v", count, err)
	}
}

func TestPurgeCommandsPreservesStoreSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.jsonl")
	link := filepath.Join(directory, "history.jsonl")
	targetStore := New(target)
	if _, err := targetStore.Import([]history.Entry{
		{Command: "export TOKEN=secret", Ordinal: 1},
		{Command: "git status", Ordinal: 2},
	}, "/tmp/history"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	removed, err := New(link).PurgeCommands(func(command string) bool {
		return strings.Contains(command, "TOKEN")
	}, true)
	if err != nil || removed != 1 {
		t.Fatalf("purge = removed %d, err %v", removed, err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("store link was replaced: info %v, err %v", info, err)
	}
	if count, err := targetStore.Count(); err != nil || count != 1 {
		t.Fatalf("target count = %d, err %v", count, err)
	}
}
