package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PrashikshitSaini/Deja/internal/render"
	"github.com/PrashikshitSaini/Deja/internal/store"
)

func TestQueryWorkerCoalescesAndPreservesSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.jsonl")
	command := "git status\n# \x1b]52;c;unsafe\x07\u202e"
	if _, err := store.New(path).Record(command, "/repo", nil, 100, 0); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		query := "missing"
		if i == 3 {
			query = "git"
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.request", i)), []byte(query+"\x00/repo\x0040\x002"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var out, errors bytes.Buffer
	code := serveQueries(dir, store.NewSearcher(path), strings.NewReader("1\n2\n3\n"), &out, &errors, time.Hour)
	if code != 0 || out.String() != "3 0\n" {
		t.Fatalf("worker: %d, %q, %q", code, out.String(), errors.String())
	}
	rows, err := os.ReadFile(filepath.Join(dir, "3.rows"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(rows), "__DEJA_META__\t2\n") || bytes.ContainsAny(rows, "\x1b\x07\u202e") {
		t.Fatalf("unsafe or missing rows: %q", rows)
	}
	for _, row := range strings.Split(strings.TrimSpace(string(rows)), "\n")[1:] {
		if len([]rune(row)) > 40 {
			t.Fatalf("width ignored: %q", row)
		}
	}
	selected, err := render.ReadResult(filepath.Join(dir, "3.json"), 1)
	if err != nil || selected != command {
		t.Fatalf("selection %q, %v", selected, err)
	}
	for i := 1; i <= 3; i++ {
		if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("%d.request", i))); !os.IsNotExist(err) {
			t.Fatalf("request %d was not cleaned up", i)
		}
	}
}

func TestQueryWorkerRejectsPublicOrSymlinkDirectory(t *testing.T) {
	dir := t.TempDir()
	public := filepath.Join(dir, "public")
	if err := os.Mkdir(public, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{public, link} {
		var out, errors bytes.Buffer
		if code := runQueryWorker([]string{"--state-dir", path}, strings.NewReader(""), &out, &errors); code == 0 {
			t.Fatalf("accepted unsafe directory %q", path)
		}
	}
}
