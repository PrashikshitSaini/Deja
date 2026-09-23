package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PrashikshitSaini/Deja/internal/store"
)

// query-worker is the private Zsh transport. Only small numeric notifications
// travel through pipes; queries and completed results live in the private
// session directory. Each response has its own ID so a late response cannot
// replace the command associated with the currently displayed palette.
func runQueryWorker(arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	set := flagSet("deja query-worker", stderr)
	stateDir := set.String("state-dir", "", "private shell session directory")
	storePath := set.String("store", "", "Deja event store")
	if err := set.Parse(arguments); err != nil {
		return 2
	}
	info, err := os.Lstat(*stateDir)
	if err != nil {
		return fail(stderr, err)
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm() != 0o700 || !ok || int(owner.Uid) != os.Geteuid() {
		return fail(stderr, fmt.Errorf("query worker requires an owned, mode-0700 directory"))
	}
	return serveQueries(*stateDir, store.NewSearcher(*storePath), stdin, stdout, stderr, 50*time.Millisecond)
}

func serveQueries(directory string, searcher querySearcher, stdin io.Reader, stdout, stderr io.Writer, delay time.Duration) int {
	requests := make(chan string, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		defer close(requests)
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			id := scanner.Text()
			if _, err := strconv.ParseUint(id, 10, 64); err != nil {
				continue
			}
			// Coalesce queued work even while a slow history reload is running.
			select {
			case old := <-requests:
				os.Remove(filepath.Join(directory, old+".request"))
			default:
			}
			select {
			case requests <- id:
			case <-done:
				return
			}
		}
	}()
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()
	var pending string
	for {
		select {
		case id, open := <-requests:
			if !open {
				if pending != "" {
					return answerQuery(directory, pending, searcher, stdout, stderr)
				}
				return 0
			}
			if pending != "" {
				os.Remove(filepath.Join(directory, pending+".request"))
			}
			pending = id
			timer.Reset(delay)
		case <-timer.C:
			if answerQuery(directory, pending, searcher, stdout, stderr) != 0 {
				return 1
			}
			pending = ""
		}
	}
}

func answerQuery(directory, id string, searcher querySearcher, stdout, stderr io.Writer) int {
	prefix := filepath.Join(directory, id)
	request, err := os.ReadFile(prefix + ".request")
	os.Remove(prefix + ".request")
	code := 1
	var rows bytes.Buffer
	if err == nil {
		fields := strings.Split(string(request), "\x00")
		if len(fields) == 4 {
			args := []string{"--query-stdin", "--cwd", fields[1], "--width", fields[2],
				"--format", "zle", "--zle-meta", "--results-file", prefix + ".json"}
			if fields[3] != "" {
				args = append(args, "--visible-rows", fields[3])
			}
			code = runQueryWithSearcher(args, strings.NewReader(fields[0]), &rows, stderr, searcher)
		}
	}
	if err := os.WriteFile(prefix+".rows", rows.Bytes(), 0o600); err != nil {
		code = 1
	}
	if _, err := fmt.Fprintf(stdout, "%s %d\n", id, code); err != nil {
		return 1
	}
	return 0
}
