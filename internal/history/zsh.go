package history

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var extendedHistory = regexp.MustCompile(`(?s)^: ([0-9]+):([0-9]+);(.*)$`)

type Entry struct {
	Command    string
	OccurredAt float64
	Duration   float64
	Ordinal    int
}

type zshRecord struct {
	lines []string
	entry Entry
}

func continuedLine(line string) bool {
	backslashes := 0
	for index := len(line) - 1; index >= 0 && line[index] == '\\'; index-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func parseEntry(raw string, ordinal int) (Entry, error) {
	entry := Entry{Command: raw, Ordinal: ordinal}
	if groups := extendedHistory.FindStringSubmatch(raw); groups != nil {
		timestamp, timestampErr := strconv.ParseFloat(groups[1], 64)
		duration, durationErr := strconv.ParseFloat(groups[2], 64)
		if timestampErr != nil || durationErr != nil {
			return Entry{}, fmt.Errorf("invalid extended history entry %d", ordinal)
		}
		entry.Command = strings.TrimSpace(groups[3])
		entry.OccurredAt = timestamp
		entry.Duration = duration
	}
	return entry, nil
}

// ReadZsh reads plain and EXTENDED_HISTORY-formatted Zsh history files.
func ReadZsh(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var entries []Entry
	var pending []string
	ordinal := 0

	consume := func(raw string) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		ordinal++
		entry, err := parseEntry(raw, ordinal)
		if err != nil {
			return err
		}
		if entry.Command != "" {
			entries = append(entries, entry)
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if continuedLine(line) {
			pending = append(pending, strings.TrimSuffix(line, `\`))
			continue
		}
		if len(pending) > 0 {
			pending = append(pending, line)
			if err := consume(strings.Join(pending, "\n")); err != nil {
				return nil, err
			}
			pending = nil
		} else if err := consume(line); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(pending) > 0 {
		if err := consume(strings.Join(pending, "\n")); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func readZshRecords(path string) ([]zshRecord, bool, []byte, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, false, nil, err
	}
	content := string(contentBytes)
	finalNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if finalNewline {
		lines = lines[:len(lines)-1]
	}
	records := make([]zshRecord, 0, len(lines))
	pending := make([]string, 0, 2)
	ordinal := 0
	finish := func() error {
		if len(pending) == 0 {
			return nil
		}
		parsedLines := append([]string(nil), pending...)
		for index := range parsedLines {
			if continuedLine(parsedLines[index]) {
				parsedLines[index] = strings.TrimSuffix(parsedLines[index], `\`)
			}
		}
		raw := strings.TrimSpace(strings.Join(parsedLines, "\n"))
		record := zshRecord{lines: append([]string(nil), pending...)}
		if raw != "" {
			ordinal++
			entry, err := parseEntry(raw, ordinal)
			if err != nil {
				return err
			}
			record.entry = entry
		}
		records = append(records, record)
		pending = pending[:0]
		return nil
	}
	for _, line := range lines {
		pending = append(pending, line)
		if continuedLine(line) {
			continue
		}
		if err := finish(); err != nil {
			return nil, false, nil, err
		}
	}
	if err := finish(); err != nil {
		return nil, false, nil, err
	}
	return records, finalNewline, contentBytes, nil
}

// PurgeZsh counts matching records and atomically removes them when apply is
// true. Nonmatching records retain their original Zsh history representation.
func PurgeZsh(path string, match func(command string) bool, apply bool) (int, error) {
	if match == nil {
		return 0, fmt.Errorf("purge matcher is required")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return 0, err
	}
	path = resolved
	records, finalNewline, original, err := readZshRecords(path)
	if err != nil {
		return 0, err
	}
	removed := 0
	kept := make([]zshRecord, 0, len(records))
	for _, record := range records {
		if record.entry.Command != "" && match(record.entry.Command) {
			removed++
			continue
		}
		kept = append(kept, record)
	}
	if !apply || removed == 0 {
		return removed, nil
	}
	parent := filepath.Dir(path)
	temporary, err := os.CreateTemp(parent, ".deja-zsh-history-*")
	if err != nil {
		return 0, err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return 0, err
	}
	writer := bufio.NewWriter(temporary)
	lineIndex := 0
	totalLines := 0
	for _, record := range kept {
		totalLines += len(record.lines)
	}
	for _, record := range kept {
		for _, line := range record.lines {
			lineIndex++
			if _, err := writer.WriteString(line); err != nil {
				temporary.Close()
				return 0, err
			}
			if lineIndex < totalLines || finalNewline {
				if err := writer.WriteByte('\n'); err != nil {
					temporary.Close()
					return 0, err
				}
			}
		}
	}
	if err := writer.Flush(); err != nil {
		temporary.Close()
		return 0, err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return 0, err
	}
	if err := temporary.Close(); err != nil {
		return 0, err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if !bytes.Equal(current, original) {
		return 0, fmt.Errorf("%s changed during purge; retry after closing other active shells", path)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return 0, err
	}
	return removed, nil
}
