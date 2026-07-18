package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareRedactsGitCommitMessagesForDisplayAndInsertion(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(
		`git commit -m "feat: private details" --amend --message='second detail'`, "git",
	)
	if !visible {
		t.Fatal("git commit unexpectedly hidden")
	}
	if insert != `git commit -m "" --amend --message=""` {
		t.Fatalf("insert = %q", insert)
	}
	if display != `git commit -m <message> --amend --message=<message>` {
		t.Fatalf("display = %q", display)
	}
}

func TestPrepareRedactsCombinedGitCommitMessageFlag(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(`git commit -am "private details"`, "git")
	if !visible || insert != `git commit -am ""` || display != `git commit -am <message>` {
		t.Fatalf("Prepare() = insert %q, display %q, visible %t", insert, display, visible)
	}
}

func TestPrepareAppliesMultipleRedactionsWithoutRevealingEarlierValues(t *testing.T) {
	settings := Defaults()
	settings.Commands.RedactFlagValues = append(settings.Commands.RedactFlagValues, RedactFlagValues{
		CommandPrefix:      "git commit",
		Flags:              []string{"--author"},
		DisplayPlaceholder: "<author>",
		InsertPlaceholder:  `""`,
	})
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(
		`git commit -m "private message" --author "Private Person"`, "git",
	)
	if !visible || insert != `git commit -m "" --author ""` {
		t.Fatalf("insert = %q, visible = %t", insert, visible)
	}
	if display != `git commit -m <message> --author <author>` {
		t.Fatalf("display = %q", display)
	}
}

func TestPrepareSupportsAllowlistAndHideRules(t *testing.T) {
	settings := Defaults()
	settings.Commands.OnlyFamilies = []string{"git", "docker"}
	settings.Commands.HiddenPrefixes = []string{"git push --force"}
	settings.Commands.HiddenPatterns = []string{`(?i)--password(?:=|\s)`}
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, _, visible := settings.Prepare("kubectl get pods", "kubectl"); visible {
		t.Fatal("family outside allowlist remained visible")
	}
	if _, _, visible := settings.Prepare("git push --force origin main", "git"); visible {
		t.Fatal("hidden prefix remained visible")
	}
	if _, _, visible := settings.Prepare("docker login --password secret", "docker"); visible {
		t.Fatal("hidden regex remained visible")
	}
	if _, _, visible := settings.Prepare("git status", "git"); !visible {
		t.Fatal("allowed command was hidden")
	}
}

func TestLoadRejectsUnknownSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"mystery":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsMissingExplicitSettingsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	_, _, err := Load(path)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestDefaultPathPrefersDejaConfigAndSupportsLegacyName(t *testing.T) {
	t.Setenv("DEJA_CONFIG", "/new/deja.json")
	t.Setenv("DEZA_CONFIG", "/legacy/deza.json")
	if got := DefaultPath(); got != "/new/deja.json" {
		t.Fatalf("DefaultPath() = %q", got)
	}

	t.Setenv("DEJA_CONFIG", "")
	if got := DefaultPath(); got != "/legacy/deza.json" {
		t.Fatalf("legacy DefaultPath() = %q", got)
	}
}
