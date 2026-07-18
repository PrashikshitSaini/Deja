package matcher

import (
	"testing"

	"github.com/PrashikshitSaini/Deja/internal/model"
)

func TestWordsToleratesQuotesAndPartialInput(t *testing.T) {
	got := Words(`git commit -m "hello world"`)
	want := []string{"git", "commit", "-m", "hello world"}
	if len(got) != len(want) {
		t.Fatalf("Words() = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("Words()[%d] = %q, want %q", index, got[index], want[index])
		}
	}

	partial := Words(`ffmpeg -i "unfinished`)
	if len(partial) != 3 || partial[2] != "unfinished" {
		t.Fatalf("partial Words() = %#v", partial)
	}

	empty := Words(`git commit -m "" --allow-empty`)
	if len(empty) != 5 || empty[3] != "" {
		t.Fatalf("empty argument Words() = %#v", empty)
	}
}

func TestFamilySupportsArbitraryCommandsAndWrappers(t *testing.T) {
	tests := map[string]string{
		"docker compose up -d":              "docker",
		"git status":                        "git",
		"kubectl get pods":                  "kubectl",
		"ffmpeg -i in.mov out.mp4":          "ffmpeg",
		"aws s3 ls":                         "aws",
		"cd ~/Projects":                     "cd",
		"ls -la":                            "ls",
		"sudo -u root git status":           "git",
		"env FOO=bar command docker ps":     "docker",
		"DEBUG=1 /opt/homebrew/bin/go test": "go",
	}
	for command, want := range tests {
		if got := Family(command); got != want {
			t.Errorf("Family(%q) = %q, want %q", command, got, want)
		}
	}
}

func TestRankPrefersCurrentDirectoryAndSuccessfulFrequentVariants(t *testing.T) {
	candidates := []model.Candidate{
		{Command: "git status", Family: "git", Uses: 3, LastRun: 900, Successes: 3, StatusCount: 3, CWDHits: 0},
		{Command: "git status --short", Family: "git", Uses: 10, LastRun: 900, Successes: 10, StatusCount: 10, CWDHits: 5},
		{Command: "git commit --amend", Family: "git", Uses: 20, LastRun: 900},
	}
	ranked := Rank(candidates, "git st", "/repo", 6, 1_000)
	if len(ranked) != 2 {
		t.Fatalf("Rank() returned %d candidates, want 2", len(ranked))
	}
	if ranked[0].Command != "git status --short" {
		t.Fatalf("first candidate = %q", ranked[0].Command)
	}
}
