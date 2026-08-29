package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PrashikshitSaini/Deja/internal/matcher"
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

func TestPrepareRedactsSensitiveEnvironmentAssignments(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(
		`export OPENAI_API_KEY="sk-private" GITHUB_TOKEN=ghp_private NODE_ENV=development`, "export",
	)
	if !visible {
		t.Fatal("export command unexpectedly hidden")
	}
	if insert != `export OPENAI_API_KEY="" GITHUB_TOKEN="" NODE_ENV=development` {
		t.Fatalf("insert = %q", insert)
	}
	if display != `export OPENAI_API_KEY=<redacted> GITHUB_TOKEN=<redacted> NODE_ENV=development` {
		t.Fatalf("display = %q", display)
	}
	if strings.Contains(insert, "private") || strings.Contains(display, "private") {
		t.Fatal("redacted value leaked")
	}
}

func TestPrepareRedactsSensitiveAssignmentBeforeCommand(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(`AWS_SECRET_ACCESS_KEY=private aws s3 ls`, "aws")
	if !visible || insert != `AWS_SECRET_ACCESS_KEY="" aws s3 ls` {
		t.Fatalf("Prepare() = insert %q, display %q, visible %t", insert, display, visible)
	}
	if display != `AWS_SECRET_ACCESS_KEY=<redacted> aws s3 ls` {
		t.Fatalf("display = %q", display)
	}
}

func TestPrepareRedactsZshAppendAndIndexedAssignments(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	insert, display, visible := settings.Prepare(`export TOKEN+=private`, "export")
	if !visible || insert != `export TOKEN+=""` {
		t.Fatalf("Prepare() = insert %q, display %q, visible %t", insert, display, visible)
	}
	if display != `export TOKEN+=<redacted>` {
		t.Fatalf("display = %q", display)
	}
	for _, command := range []string{
		`typeset PRIVATE_KEY[service]=secret`,
		`typeset PRIVATE_KEY['service name']=secret`,
		`typeset PRIVATE_KEY[service\]name]=secret`,
	} {
		insert, display, visible = settings.Prepare(command, "typeset")
		if visible || insert != "" || display != "" {
			t.Fatalf("indexed secret %q remained visible: insert %q, display %q", command, insert, display)
		}
	}
}

func TestPreparePreservesShellSyntaxAroundEnvironmentRedaction(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	command := `TOKEN=private command "$URL" && printf '%s\n' *.txt >output`
	insert, display, visible := settings.Prepare(command, "command")
	if !visible {
		t.Fatal("command unexpectedly hidden")
	}
	if want := `TOKEN="" command "$URL" && printf '%s\n' *.txt >output`; insert != want {
		t.Fatalf("insert = %q, want %q", insert, want)
	}
	if want := `TOKEN=<redacted> command "$URL" && printf '%s\n' *.txt >output`; display != want {
		t.Fatalf("display = %q, want %q", display, want)
	}
}

func TestPreparePreservesAttachedRedirectionAndQuotedAssignments(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		command string
		insert  string
		display string
	}{
		{
			command: `TOKEN=private>output command "$URL"`,
			insert:  `TOKEN="">output command "$URL"`,
			display: `TOKEN=<redacted>>output command "$URL"`,
		},
		{
			command: `export 'TOKEN=private'`,
			insert:  `export TOKEN=""`,
			display: `export TOKEN=<redacted>`,
		},
	} {
		insert, display, visible := settings.Prepare(test.command, matcher.Family(test.command))
		if !visible || insert != test.insert || display != test.display {
			t.Fatalf("Prepare(%q) = insert %q, display %q, visible %t", test.command, insert, display, visible)
		}
	}
	if insert, display, visible := settings.Prepare(`export TOKEN\=private`, "export"); visible || insert != "" || display != "" {
		t.Fatalf("escaped assignment remained visible: insert %q, display %q", insert, display)
	}
}

func TestPrepareHidesDynamicSensitiveAssignmentsInsteadOfLeakingTokens(t *testing.T) {
	settings := Defaults()
	if err := settings.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		`export API_TOKEN=$(printf sk-live)`,
		`export API_TOKEN=prefix$(printf sk-live)`,
		`export API_TOKEN=prefix$((40 + 2))`,
		"export API_TOKEN=`printf sk-live`",
		`typeset PRIVATE_KEY=(private value)`,
		`export PRIVATE_KEY=<(printf private)`,
	} {
		insert, display, visible := settings.Prepare(command, matcher.Family(command))
		if visible || insert != "" || display != "" {
			t.Fatalf("dynamic secret %q remained visible: insert %q, display %q", command, insert, display)
		}
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
