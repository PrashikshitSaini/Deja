package history

import (
	"bufio"
	"fmt"
	"os"
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
		entry := Entry{Command: raw, Ordinal: ordinal}
		if groups := extendedHistory.FindStringSubmatch(raw); groups != nil {
			timestamp, timestampErr := strconv.ParseFloat(groups[1], 64)
			duration, durationErr := strconv.ParseFloat(groups[2], 64)
			if timestampErr != nil || durationErr != nil {
				return fmt.Errorf("invalid extended history entry %d", ordinal)
			}
			entry.Command = strings.TrimSpace(groups[3])
			entry.OccurredAt = timestamp
			entry.Duration = duration
		}
		if entry.Command != "" {
			entries = append(entries, entry)
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, `\`) {
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
