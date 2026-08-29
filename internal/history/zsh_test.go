package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadZshPlainExtendedAndMultiline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	content := "git status\n: 1700000000:4;docker compose up -d\n" +
		": 1700000001:1;ffmpeg -i input.mov \\\noutput.mp4\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadZsh(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("ReadZsh() returned %d entries", len(entries))
	}
	if entries[1].OccurredAt != 1_700_000_000 || entries[1].Duration != 4 {
		t.Fatalf("extended entry = %#v", entries[1])
	}
	if entries[2].Command != "ffmpeg -i input.mov \noutput.mp4" {
		t.Fatalf("multiline command = %q", entries[2].Command)
	}
}

func TestPurgeZshDryRunAndPreservesUnmatchedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	content := "git status\n" +
		": 1700000000:1;export OPENAI_API_KEY=private\n" +
		": 1700000001:1;ffmpeg -i input.mov \\\noutput.mp4\n" +
		": 1700000002:1;echo PRIVATE_TOKEN=secret \\\ncontinued\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	match := func(command string) bool {
		return strings.Contains(command, "OPENAI_API_KEY") || strings.Contains(command, "PRIVATE_TOKEN")
	}
	removed, err := PurgeZsh(path, match, false)
	if err != nil || removed != 2 {
		t.Fatalf("dry run = removed %d, err %v", removed, err)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterDryRun) != content {
		t.Fatal("dry run changed history")
	}
	removed, err = PurgeZsh(path, match, true)
	if err != nil || removed != 2 {
		t.Fatalf("purge = removed %d, err %v", removed, err)
	}
	want := "git status\n: 1700000001:1;ffmpeg -i input.mov \\\noutput.mp4\n"
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != want {
		t.Fatalf("purged history = %q, want %q", after, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("purged history permissions = %o", info.Mode().Perm())
	}
}

func TestPurgeZshDoesNotJoinLineEndingInTwoBackslashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	content := "echo literal " + `\\` + "\nexport TOKEN=secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeZsh(path, func(command string) bool {
		return strings.Contains(command, "TOKEN")
	}, true)
	if err != nil || removed != 1 {
		t.Fatalf("purge = removed %d, err %v", removed, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "echo literal " + `\\` + "\n"; string(after) != want {
		t.Fatalf("purged history = %q, want %q", after, want)
	}
}

func TestPurgeZshPreservesSymlink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target-history")
	link := filepath.Join(directory, "history")
	if err := os.WriteFile(target, []byte("export TOKEN=secret\ngit status\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeZsh(link, func(command string) bool {
		return strings.Contains(command, "TOKEN")
	}, true)
	if err != nil || removed != 1 {
		t.Fatalf("purge = removed %d, err %v", removed, err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("history link was replaced: info %v, err %v", info, err)
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != "git status\n" {
		t.Fatalf("target history = %q", after)
	}
}
