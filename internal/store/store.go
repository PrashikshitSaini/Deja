package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PrashikshitSaini/Deja/internal/history"
	"github.com/PrashikshitSaini/Deja/internal/matcher"
	"github.com/PrashikshitSaini/Deja/internal/model"
)

type Store struct {
	Path string
}

type SearchOptions struct {
	Limit       int
	MinimumUses int
	MaxAgeDays  int
	Prepare     func(command, family string) (insert, display string, visible bool)
}

func DefaultPath() string {
	if configured := os.Getenv("DEJA_STORE"); configured != "" {
		return configured
	}
	// DEZA_STORE is retained for one rename-compatible release. New setups
	// should use DEJA_STORE.
	if configured := os.Getenv("DEZA_STORE"); configured != "" {
		return configured
	}
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "deja", "history.jsonl")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".deja-history.jsonl")
	}
	return filepath.Join(home, ".local", "share", "deja", "history.jsonl")
}

func New(path string) Store {
	if path == "" {
		path = DefaultPath()
	}
	return Store{Path: path}
}

func hash(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func liveIdentity(command string, occurredAt float64) string {
	if occurredAt <= 0 {
		return "live:" + hash(command, strconv.FormatInt(time.Now().UnixNano(), 10))
	}
	// Truncated to whole seconds so a live-recorded event deduplicates
	// against the same entry later re-imported from EXTENDED_HISTORY,
	// which stores integer timestamps.
	return "run:" + hash(command, strconv.FormatInt(int64(occurredAt), 10))
}

func purgeInvocation(command string) bool {
	words := matcher.Words(command)
	segmentStart := 0
	for segmentEnd := 0; segmentEnd <= len(words); segmentEnd++ {
		atBoundary := segmentEnd == len(words)
		if !atBoundary {
			atBoundary = len(words[segmentEnd]) == 1 && strings.Contains("|&;()", words[segmentEnd])
		}
		if !atBoundary {
			continue
		}
		segment := words[segmentStart:segmentEnd]
		for len(segment) > 0 {
			prefix := strings.ToLower(segment[0])
			if prefix != "if" && prefix != "then" && prefix != "elif" && prefix != "else" &&
				prefix != "while" && prefix != "until" && prefix != "do" && prefix != "!" && prefix != "{" {
				break
			}
			segment = segment[1:]
		}
		if matcher.Family(strings.Join(segment, " ")) == "deja" {
			for index := 0; index+1 < len(segment); index++ {
				if strings.EqualFold(filepath.Base(strings.TrimPrefix(segment[index], `\`)), "deja") &&
					strings.EqualFold(segment[index+1], "purge") {
					return true
				}
			}
		}
		segmentStart = segmentEnd + 1
	}
	return false
}

func (store Store) resolved() (Store, error) {
	path, err := filepath.EvalSymlinks(store.Path)
	if err == nil {
		store.Path = path
		return store, nil
	}
	if os.IsNotExist(err) {
		return store, nil
	}
	return Store{}, err
}

func (store Store) ensureParent() error {
	parent := filepath.Dir(store.Path)
	_, statErr := os.Stat(parent)
	parentExisted := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if !parentExisted {
		return os.Chmod(parent, 0o700)
	}
	return nil
}

func (store Store) withLock(exclusive bool, action func() error) error {
	if err := store.ensureParent(); err != nil {
		return err
	}
	lock, err := os.OpenFile(store.Path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	operation := syscall.LOCK_SH
	if exclusive {
		operation = syscall.LOCK_EX
	}
	if err := syscall.Flock(int(lock.Fd()), operation); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) //nolint:errcheck
	return action()
}

func (store Store) loadUnlocked() ([]model.Event, error) {
	file, err := os.Open(store.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	events := make([]model.Event, 0)
	seen := make(map[string]bool)
	line := 0
	for scanner.Scan() {
		line++
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var event model.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("decode %s line %d: %w", store.Path, line, err)
		}
		if event.ID == "" || seen[event.ID] {
			continue
		}
		seen[event.ID] = true
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (store Store) addUnique(events []model.Event) (int, error) {
	var err error
	store, err = store.resolved()
	if err != nil {
		return 0, err
	}
	added := 0
	err = store.withLock(true, func() error {
		existing, err := store.loadUnlocked()
		if err != nil {
			return err
		}
		seen := make(map[string]bool, len(existing))
		for _, event := range existing {
			seen[event.ID] = true
		}

		file, err := os.OpenFile(store.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := os.Chmod(store.Path, 0o600); err != nil {
			return err
		}
		writer := bufio.NewWriter(file)
		encoder := json.NewEncoder(writer)
		for _, event := range events {
			if event.ID == "" || seen[event.ID] {
				continue
			}
			if err := encoder.Encode(event); err != nil {
				return err
			}
			seen[event.ID] = true
			added++
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		return file.Sync()
	})
	return added, err
}

func (store Store) replaceUnlocked(events []model.Event) error {
	parent := filepath.Dir(store.Path)
	temporary, err := os.CreateTemp(parent, ".deja-store-*.jsonl")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	writer := bufio.NewWriter(temporary)
	encoder := json.NewEncoder(writer)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			temporary.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, store.Path)
}

// PurgeCommands counts matching events and atomically removes them when apply
// is true. The caller controls matching so exact and substring purges share the
// same locked rewrite path.
func (store Store) PurgeCommands(match func(command string) bool, apply bool) (int, error) {
	if match == nil {
		return 0, fmt.Errorf("purge matcher is required")
	}
	targetStore, err := store.resolved()
	if err != nil {
		return 0, err
	}
	removed := 0
	err = targetStore.withLock(apply, func() error {
		events, err := targetStore.loadUnlocked()
		if err != nil {
			return err
		}
		kept := make([]model.Event, 0, len(events))
		for _, event := range events {
			if match(event.Command) {
				removed++
				continue
			}
			kept = append(kept, event)
		}
		if !apply || removed == 0 {
			return nil
		}
		return targetStore.replaceUnlocked(kept)
	})
	return removed, err
}

func (store Store) Import(entries []history.Entry, source string) (int, error) {
	events := make([]model.Event, 0, len(entries))
	for _, entry := range entries {
		family := matcher.Family(entry.Command)
		if family == "" || purgeInvocation(entry.Command) {
			continue
		}
		identity := ""
		if entry.OccurredAt > 0 {
			identity = liveIdentity(entry.Command, entry.OccurredAt)
		} else {
			identity = "import:" + hash(source, strconv.Itoa(entry.Ordinal), entry.Command)
		}
		events = append(events, model.Event{
			ID: identity, Command: entry.Command, Family: family,
			OccurredAt: entry.OccurredAt, Duration: entry.Duration, Source: "zsh-history",
		})
	}
	return store.addUnique(events)
}

func (store Store) Record(command, cwd string, exitStatus *int, occurredAt, duration float64) (bool, error) {
	command = strings.TrimSpace(command)
	family := matcher.Family(command)
	if command == "" || family == "" {
		return false, nil
	}
	if purgeInvocation(command) {
		return false, nil
	}
	if occurredAt <= 0 {
		occurredAt = float64(time.Now().UnixNano()) / 1e9
	}
	added, err := store.addUnique([]model.Event{{
		ID: liveIdentity(command, occurredAt), Command: command, Family: family,
		CWD: cwd, ExitStatus: exitStatus, OccurredAt: occurredAt,
		Duration: duration, Source: "zsh-live",
	}})
	return added == 1, err
}

func (store Store) Search(query, cwd string, limit int) ([]model.Candidate, error) {
	return store.SearchWithOptions(query, cwd, SearchOptions{Limit: limit})
}

func (store Store) SearchWithOptions(query, cwd string, options SearchOptions) ([]model.Candidate, error) {
	if matcher.Family(query) == "" {
		return nil, nil
	}
	var err error
	store, err = store.resolved()
	if err != nil {
		return nil, err
	}
	var events []model.Event
	if err := store.withLock(false, func() error {
		var err error
		events, err = store.loadUnlocked()
		return err
	}); err != nil {
		return nil, err
	}

	familyPrefix := matcher.Family(query)
	exactFamilyExists := false
	for _, event := range events {
		if event.Family == familyPrefix {
			exactFamilyExists = true
			break
		}
	}
	byCommand := make(map[string]*model.Candidate)
	oldestAllowed := float64(0)
	if options.MaxAgeDays > 0 {
		oldestAllowed = float64(time.Now().Add(-time.Duration(options.MaxAgeDays) * 24 * time.Hour).Unix())
	}
	for _, event := range events {
		if exactFamilyExists && event.Family != familyPrefix {
			continue
		}
		if !exactFamilyExists && !strings.HasPrefix(event.Family, familyPrefix) {
			continue
		}
		if oldestAllowed > 0 && event.OccurredAt > 0 && event.OccurredAt < oldestAllowed {
			continue
		}
		insert, display, visible := event.Command, "", true
		if options.Prepare != nil {
			insert, display, visible = options.Prepare(event.Command, event.Family)
		}
		if !visible || strings.TrimSpace(insert) == "" {
			continue
		}
		key := insert + "\x00" + display
		candidate := byCommand[key]
		if candidate == nil {
			candidate = &model.Candidate{Command: insert, Display: display, Family: event.Family}
			byCommand[key] = candidate
		}
		candidate.Uses++
		if event.OccurredAt > candidate.LastRun {
			candidate.LastRun = event.OccurredAt
		}
		if event.ExitStatus != nil {
			candidate.StatusCount++
			if *event.ExitStatus == 0 {
				candidate.Successes++
			}
		}
		if cwd != "" && event.CWD == cwd {
			candidate.CWDHits++
		}
	}
	candidates := make([]model.Candidate, 0, len(byCommand))
	for _, candidate := range byCommand {
		if options.MinimumUses > 1 && candidate.Uses < options.MinimumUses {
			continue
		}
		candidates = append(candidates, *candidate)
	}
	return matcher.Rank(candidates, query, cwd, options.Limit, float64(time.Now().UnixNano())/1e9), nil
}

func (store Store) Count() (int, error) {
	var err error
	store, err = store.resolved()
	if err != nil {
		return 0, err
	}
	count := 0
	err = store.withLock(false, func() error {
		events, err := store.loadUnlocked()
		count = len(events)
		return err
	})
	return count, err
}
