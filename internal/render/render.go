package render

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/PrashikshitSaini/Deja/internal/matcher"
	"github.com/PrashikshitSaini/Deja/internal/model"
	"github.com/PrashikshitSaini/Deja/internal/theme"
)

const reset = "\x1b[0m"

type Options struct {
	Color                bool
	RankSymbols          bool
	Width                int
	RankColors           []string
	FamilyColor          string
	DifferenceColor      string
	MetadataColor        string
	ShowUsage            bool
	ShowSuccessRate      bool
	ShowCurrentDirectory bool
}

func DefaultOptions(color bool) Options {
	return Options{
		Color:                color,
		RankColors:           []string{"default"},
		FamilyColor:          "inherit",
		DifferenceColor:      "bright-yellow",
		MetadataColor:        "dim",
		ShowUsage:            true,
		ShowSuccessRate:      true,
		ShowCurrentDirectory: true,
	}
}

type span struct {
	start int
	end   int
}

func tokenSpans(command string) []span {
	spans := make([]span, 0)
	start := -1
	var quote byte
	escaped := false
	for index := 0; index < len(command); index++ {
		character := command[index]
		if start < 0 {
			if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
				continue
			}
			start = index
		}
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			spans = append(spans, span{start: start, end: index})
			start = -1
		}
	}
	if start >= 0 {
		spans = append(spans, span{start: start, end: len(command)})
	}
	return spans
}

func differingPositions(candidates []model.Candidate) map[int]bool {
	differing := make(map[int]bool)
	rows := make([][]string, len(candidates))
	width := 0
	for index, candidate := range candidates {
		rows[index] = matcher.Words(safeDisplay(candidate.DisplayCommand()))
		if len(rows[index]) > width {
			width = len(rows[index])
		}
	}
	if len(rows) == 1 {
		for position := 1; position < len(rows[0]); position++ {
			differing[position] = true
		}
		return differing
	}
	for position := 1; position < width; position++ {
		values := make(map[string]bool)
		for _, row := range rows {
			value := "\x00missing"
			if position < len(row) {
				value = row[position]
			}
			values[value] = true
		}
		if len(values) > 1 {
			differing[position] = true
		}
	}
	return differing
}

func singleLine(command string) string {
	parts := strings.Split(command, "\n")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return strings.Join(parts, " ↵ ")
}

func bidiControl(character rune) bool {
	return character == '\u061c' || character == '\u200e' || character == '\u200f' ||
		(character >= '\u202a' && character <= '\u202e') ||
		(character >= '\u2066' && character <= '\u2069')
}

func safeDisplay(command string) string {
	command = singleLine(command)
	var output strings.Builder
	for _, character := range command {
		if unicode.IsControl(character) || bidiControl(character) {
			if character <= 0xffff {
				fmt.Fprintf(&output, `\u%04X`, character)
			} else {
				fmt.Fprintf(&output, `\U%08X`, character)
			}
			continue
		}
		output.WriteRune(character)
	}
	return output.String()
}

type cellRole uint8

const (
	roleBase cellRole = iota
	roleFamily
	roleDifference
	roleMetadata
)

type cell struct {
	value rune
	role  cellRole
}

func appendCells(cells []cell, value string, role cellRole) []cell {
	for _, character := range value {
		cells = append(cells, cell{value: character, role: role})
	}
	return cells
}

func commandCells(command string, differing map[int]bool, color bool) []cell {
	display := safeDisplay(command)
	spans := tokenSpans(display)
	cells := make([]cell, 0, len([]rune(display)))
	cursor := 0
	for position, token := range spans {
		cells = appendCells(cells, display[cursor:token.start], roleBase)
		value := display[token.start:token.end]
		if color {
			if position == 0 {
				cells = appendCells(cells, value, roleFamily)
			} else if differing[position] {
				cells = appendCells(cells, value, roleDifference)
			} else {
				cells = appendCells(cells, value, roleBase)
			}
		} else if differing[position] {
			cells = appendCells(cells, "["+value+"]", roleBase)
		} else {
			cells = appendCells(cells, value, roleBase)
		}
		cursor = token.end
	}
	return appendCells(cells, display[cursor:], roleBase)
}

func shortenCells(cells []cell, maximum int) []cell {
	if maximum <= 0 || len(cells) <= maximum {
		return cells
	}
	marker := []cell{}
	marker = appendCells(marker, " ... ", roleBase)
	if maximum <= len(marker)+2 {
		return cells[:maximum]
	}
	available := maximum - len(marker)
	headTarget := available * 3 / 5
	tailTarget := available - headTarget
	headEnd := headTarget
	for headEnd > 1 && !unicode.IsSpace(cells[headEnd-1].value) {
		headEnd--
	}
	for headEnd > 0 && unicode.IsSpace(cells[headEnd-1].value) {
		headEnd--
	}
	tailStart := len(cells) - tailTarget
	originalTailStart := tailStart
	for tailStart < len(cells)-1 && !unicode.IsSpace(cells[tailStart].value) {
		tailStart++
	}
	if tailStart == len(cells)-1 && !unicode.IsSpace(cells[tailStart].value) {
		// The final argument is wider than the tail budget. Its last characters
		// are more useful than a lone closing quote or bracket.
		tailStart = originalTailStart
	}
	for tailStart < len(cells) && unicode.IsSpace(cells[tailStart].value) {
		tailStart++
	}
	if headEnd == 0 || tailStart >= len(cells) {
		headEnd = headTarget
		tailStart = len(cells) - tailTarget
	}
	shortened := make([]cell, 0, maximum)
	shortened = append(shortened, cells[:headEnd]...)
	shortened = append(shortened, marker...)
	shortened = append(shortened, cells[tailStart:]...)
	if len(shortened) > maximum {
		shortened = shortened[:maximum]
	}
	return shortened
}

func styleFor(role cellRole, base string, options Options) string {
	style := "inherit"
	switch role {
	case roleFamily:
		style = options.FamilyColor
	case roleDifference:
		style = options.DifferenceColor
	case roleMetadata:
		style = options.MetadataColor
	}
	if style == "inherit" {
		return base
	}
	return style
}

func renderCells(cells []cell, base string, options Options) string {
	var output strings.Builder
	if !options.Color {
		for _, item := range cells {
			output.WriteRune(item.value)
		}
		return output.String()
	}
	active := "\x00"
	for _, item := range cells {
		style := styleFor(item.role, base, options)
		if style != active {
			output.WriteString(reset)
			output.WriteString(theme.ANSI(style))
			active = style
		}
		output.WriteRune(item.value)
	}
	output.WriteString(reset)
	return output.String()
}

func metadataCells(candidate model.Candidate, options Options) []cell {
	parts := make([]string, 0, 3)
	if options.ShowUsage {
		parts = append(parts, fmt.Sprintf("×%d", candidate.Uses))
	}
	if options.ShowSuccessRate {
		if successRate, ok := candidate.SuccessRate(); ok {
			parts = append(parts, fmt.Sprintf("%.0f%% ok", successRate*100))
		}
	}
	if options.ShowCurrentDirectory && candidate.CWDHits > 0 {
		parts = append(parts, "here")
	}
	if len(parts) == 0 {
		return nil
	}
	return appendCells(nil, strings.Join(parts, " · "), roleMetadata)
}

func Rows(candidates []model.Candidate, options Options) []string {
	differing := differingPositions(candidates)
	rows := make([]string, 0, len(candidates))
	for index, candidate := range candidates {
		command := commandCells(candidate.DisplayCommand(), differing, options.Color)
		metadata := metadataCells(candidate, options)
		separator := []cell(nil)
		if len(metadata) > 0 {
			separator = appendCells(nil, "  ", roleBase)
		}
		if options.Width > 0 {
			symbolWidth := 0
			if options.RankSymbols {
				symbolWidth = 3
			}
			commandWidth := options.Width - symbolWidth - len(separator) - len(metadata)
			if commandWidth < 1 {
				commandWidth = 1
			}
			command = shortenCells(command, commandWidth)
		}
		base := "inherit"
		if len(options.RankColors) > 0 {
			colorIndex := index
			if colorIndex >= len(options.RankColors) {
				colorIndex = len(options.RankColors) - 1
			}
			base = options.RankColors[colorIndex]
		}
		cells := make([]cell, 0, len(command)+len(separator)+len(metadata)+2)
		if options.RankSymbols {
			cells = appendCells(cells, fmt.Sprintf("%2d ", index+1), roleMetadata)
		}
		cells = append(cells, command...)
		cells = append(cells, separator...)
		cells = append(cells, metadata...)
		rows = append(rows, renderCells(cells, base, options))
	}
	return rows
}

type resultFile struct {
	Version    int               `json:"version"`
	Candidates []model.Candidate `json:"candidates"`
}

func WriteResults(path string, candidates []model.Candidate) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".deja-results-*.json")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := json.NewEncoder(temporary).Encode(resultFile{Version: 1, Candidates: candidates}); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func ReadResult(path string, index int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var results resultFile
	if err := json.NewDecoder(file).Decode(&results); err != nil {
		return "", err
	}
	if index < 1 || index > len(results.Candidates) {
		return "", fmt.Errorf("selection %d is outside 1..%d", index, len(results.Candidates))
	}
	return results.Candidates[index-1].Command, nil
}
