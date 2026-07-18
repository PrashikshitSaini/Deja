package history

import (
	"os"
	"path/filepath"
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
