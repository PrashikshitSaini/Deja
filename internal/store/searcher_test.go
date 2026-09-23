package store

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/PrashikshitSaini/Deja/internal/history"
)

func TestSessionSearchMatchesUncachedAndRefreshes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	s := New(path)
	searcher := NewSearcher(path)
	compare := func() {
		t.Helper()
		for _, query := range []string{"git", "g", "git status", "docker", "missing"} {
			for _, options := range []SearchOptions{{Limit: 6}, {Limit: 6, MinimumUses: 2}, {Limit: 6, MaxAgeDays: 1},
				{Limit: 6, Prepare: func(command, family string) (string, string, bool) {
					return strings.ReplaceAll(command, "status", "<redacted>"), "display", family != "docker"
				}}} {
				want, err := s.SearchWithOptions(query, "/repo", options)
				if err != nil {
					t.Fatal(err)
				}
				got, err := searcher.SearchWithOptions(query, "/repo", options)
				if err != nil {
					t.Fatal(err)
				}
				for i := range want {
					want[i].Score = 0
				}
				for i := range got {
					got[i].Score = 0
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("query %q: got %#v, want %#v", query, got, want)
				}
			}
		}
	}
	compare() // absent store
	for i, command := range []string{"git status", "git status", "git-lfs status", "docker ps"} {
		if _, err := s.Record(command, "/repo", intPointer(0), float64(i+100), 0); err != nil {
			t.Fatal(err)
		}
		compare()
	}
	if _, err := s.PurgeCommands(func(command string) bool { return command == "git status" }, true); err != nil {
		t.Fatal(err)
	}
	compare()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	compare()
}

func TestSessionSearchReloadsSameSizeReplacementAndRejectsCorruption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	s := New(path)
	if _, err := s.Record("git status", "", nil, 100, 0); err != nil {
		t.Fatal(err)
	}
	searcher := NewSearcher(path)
	if _, err := searcher.SearchWithOptions("git", "", SearchOptions{}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	data, _ := os.ReadFile(path)
	replacement := path + ".replacement"
	if err := os.WriteFile(replacement, []byte(strings.ReplaceAll(string(data), "git status", "git branch")), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(replacement, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	got, err := searcher.SearchWithOptions("git", "", SearchOptions{Limit: 6})
	if err != nil || len(got) != 1 || got[0].Command != "git branch" {
		t.Fatalf("replacement: %#v, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("invalid json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := searcher.SearchWithOptions("git", "", SearchOptions{}); err == nil || got != nil {
		t.Fatalf("corruption returned stale data: %#v, %v", got, err)
	}
}

func BenchmarkSessionSearch(b *testing.B) {
	for _, count := range []int{20_000, 100_000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "history.jsonl")
			entries := make([]history.Entry, count)
			for i := range entries {
				entries[i] = history.Entry{Command: fmt.Sprintf("git checkout branch-%d", i%250), OccurredAt: float64(1_700_000_000 + i), Ordinal: i + 1}
			}
			if _, err := New(path).Import(entries, "/synthetic/history"); err != nil {
				b.Fatal(err)
			}
			s := NewSearcher(path)
			if _, err := s.SearchWithOptions("git checkout", "/repo", SearchOptions{Limit: 6}); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := s.SearchWithOptions("git checkout", "/repo", SearchOptions{Limit: 6}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
