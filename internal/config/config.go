package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PrashikshitSaini/Deja/internal/matcher"
	"github.com/PrashikshitSaini/Deja/internal/theme"
)

const Version = 1

type Metadata struct {
	Usage            bool `json:"usage"`
	SuccessRate      bool `json:"success_rate"`
	CurrentDirectory bool `json:"current_directory"`
}

type Colors struct {
	Enabled    bool     `json:"enabled"`
	Rank       []string `json:"rank"`
	Family     string   `json:"family"`
	Difference string   `json:"difference"`
	Metadata   string   `json:"metadata"`
}

type Display struct {
	Limit              int      `json:"limit"`
	CandidateLimit     int      `json:"candidate_limit"`
	MinimumUses        int      `json:"minimum_uses"`
	MaximumAgeDays     int      `json:"maximum_age_days"`
	MinimumQueryLength int      `json:"minimum_query_length"`
	Metadata           Metadata `json:"metadata"`
	Colors             Colors   `json:"colors"`
}

type RedactFlagValues struct {
	CommandPrefix      string   `json:"command_prefix"`
	Flags              []string `json:"flags"`
	DisplayPlaceholder string   `json:"display_placeholder"`
	InsertPlaceholder  string   `json:"insert_placeholder"`
}

type Commands struct {
	OnlyFamilies               []string           `json:"only_families"`
	HiddenFamilies             []string           `json:"hidden_families"`
	HiddenPrefixes             []string           `json:"hidden_prefixes"`
	HiddenPatterns             []string           `json:"hidden_patterns"`
	RedactEnvironmentVariables []string           `json:"redact_environment_variables"`
	RedactFlagValues           []RedactFlagValues `json:"redact_flag_values"`
}

type Settings struct {
	Version  int      `json:"version"`
	Display  Display  `json:"display"`
	Commands Commands `json:"commands"`

	hiddenRegex            []*regexp.Regexp
	sensitiveVariableRegex []*regexp.Regexp
}

func Defaults() Settings {
	return Settings{
		Version: Version,
		Display: Display{
			Limit:              6,
			CandidateLimit:     100,
			MinimumUses:        1,
			MaximumAgeDays:     0,
			MinimumQueryLength: 1,
			Metadata: Metadata{
				Usage:            true,
				SuccessRate:      true,
				CurrentDirectory: true,
			},
			Colors: Colors{
				Enabled:    true,
				Rank:       []string{"dark-red", "red", "orange", "gold", "default", "default"},
				Family:     "inherit",
				Difference: "bright-yellow",
				Metadata:   "dim",
			},
		},
		Commands: Commands{
			HiddenFamilies: []string{"clear", "exit", "fc", "history", "logout"},
			RedactEnvironmentVariables: []string{
				`(?i)(api_?key|access_?key|token|secret|password|passwd|private_?key|client_?secret|credential)`,
			},
			RedactFlagValues: []RedactFlagValues{{
				CommandPrefix:      "git commit",
				Flags:              []string{"-m", "-am", "--message"},
				DisplayPlaceholder: "<message>",
				InsertPlaceholder:  `""`,
			}},
		},
	}
}

func DefaultPath() string {
	if configured := os.Getenv("DEJA_CONFIG"); configured != "" {
		return configured
	}
	// DEZA_CONFIG is retained for one rename-compatible release. New setups
	// should use DEJA_CONFIG.
	if configured := os.Getenv("DEZA_CONFIG"); configured != "" {
		return configured
	}
	if local, err := filepath.Abs("deja.json"); err == nil {
		if _, err := os.Stat(local); err == nil {
			return local
		}
	}
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "deja", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "deja.json")
	}
	return filepath.Join(home, ".config", "deja", "config.json")
}

func Load(path string) (Settings, string, error) {
	explicit := path != "" || os.Getenv("DEJA_CONFIG") != "" || os.Getenv("DEZA_CONFIG") != ""
	if path == "" {
		path = DefaultPath()
	}
	settings := Defaults()
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if explicit {
			return Settings{}, path, err
		}
		if err := settings.Validate(); err != nil {
			return Settings{}, path, err
		}
		return settings, path, nil
	}
	if err != nil {
		return Settings{}, path, err
	}
	defer file.Close()
	if err := decode(file, &settings); err != nil {
		return Settings{}, path, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := settings.Validate(); err != nil {
		return Settings{}, path, fmt.Errorf("validate %s: %w", path, err)
	}
	return settings, path, nil
}

func decode(reader io.Reader, settings *Settings) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(settings); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func (settings *Settings) Validate() error {
	if settings.Version != Version {
		return fmt.Errorf("version must be %d", Version)
	}
	if settings.Display.Limit < 1 || settings.Display.Limit > 50 {
		return errors.New("display.limit must be between 1 and 50")
	}
	if settings.Display.CandidateLimit < settings.Display.Limit || settings.Display.CandidateLimit > 5000 {
		return errors.New("display.candidate_limit must be between display.limit and 5000")
	}
	if settings.Display.MinimumUses < 1 {
		return errors.New("display.minimum_uses must be at least 1")
	}
	if settings.Display.MaximumAgeDays < 0 {
		return errors.New("display.maximum_age_days cannot be negative")
	}
	if settings.Display.MinimumQueryLength < 1 {
		return errors.New("display.minimum_query_length must be at least 1")
	}
	colors := append([]string{}, settings.Display.Colors.Rank...)
	colors = append(colors, settings.Display.Colors.Family, settings.Display.Colors.Difference, settings.Display.Colors.Metadata)
	for _, color := range colors {
		if !theme.Valid(color) {
			return fmt.Errorf("unknown color %q (choose one of %s)", color, strings.Join(theme.Names(), ", "))
		}
	}
	settings.hiddenRegex = settings.hiddenRegex[:0]
	for _, pattern := range settings.Commands.HiddenPatterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("commands.hidden_patterns %q: %w", pattern, err)
		}
		settings.hiddenRegex = append(settings.hiddenRegex, compiled)
	}
	settings.sensitiveVariableRegex = settings.sensitiveVariableRegex[:0]
	for _, pattern := range settings.Commands.RedactEnvironmentVariables {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("commands.redact_environment_variables %q: %w", pattern, err)
		}
		settings.sensitiveVariableRegex = append(settings.sensitiveVariableRegex, compiled)
	}
	for index, rule := range settings.Commands.RedactFlagValues {
		if len(matcher.Words(rule.CommandPrefix)) == 0 {
			return fmt.Errorf("commands.redact_flag_values[%d].command_prefix is required", index)
		}
		if len(rule.Flags) == 0 {
			return fmt.Errorf("commands.redact_flag_values[%d].flags cannot be empty", index)
		}
		if rule.DisplayPlaceholder == "" || rule.InsertPlaceholder == "" {
			return fmt.Errorf("commands.redact_flag_values[%d] placeholders cannot be empty", index)
		}
		for _, flag := range rule.Flags {
			if !strings.HasPrefix(flag, "-") {
				return fmt.Errorf("commands.redact_flag_values[%d] flag %q must start with -", index, flag)
			}
		}
	}
	return nil
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), wanted) {
			return true
		}
	}
	return false
}

func normalizedWords(value string) []string {
	words := matcher.Words(value)
	for index := range words {
		words[index] = strings.ToLower(words[index])
	}
	return words
}

func hasWordPrefix(words, prefix []string) bool {
	if len(prefix) > len(words) {
		return false
	}
	for index := range prefix {
		if words[index] != prefix[index] {
			return false
		}
	}
	return true
}

var safeShellToken = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./~-]+$`)
var environmentAssignmentName = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)(?:\+)?$`)

func shellToken(value string) string {
	if value == "" {
		return `''`
	}
	if safeShellToken.MatchString(value) || strings.Contains("|&;()", value) && len(value) == 1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func rewriteFlagValues(command string, rules []RedactFlagValues, display bool) string {
	tokens := matcher.Words(command)
	commandWords := normalizedWords(command)
	applicable := make([]RedactFlagValues, 0, len(rules))
	for _, rule := range rules {
		if hasWordPrefix(commandWords, normalizedWords(rule.CommandPrefix)) {
			applicable = append(applicable, rule)
		}
	}
	if len(applicable) == 0 {
		return command
	}
	result := make([]string, 0, len(tokens))
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		matched := false
		for _, rule := range applicable {
			placeholder := rule.InsertPlaceholder
			if display {
				placeholder = rule.DisplayPlaceholder
			}
			for _, flag := range rule.Flags {
				if token == flag {
					result = append(result, flag)
					if index+1 < len(tokens) {
						result = append(result, placeholder)
						index++
					}
					matched = true
					break
				}
				if strings.HasPrefix(token, flag+"=") {
					result = append(result, flag+"="+placeholder)
					matched = true
					break
				}
				if len(flag) == 2 && strings.HasPrefix(token, flag) && len(token) > len(flag) {
					result = append(result, flag+placeholder)
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			result = append(result, shellToken(token))
		}
	}
	return strings.Join(result, " ")
}

func rewriteEnvironmentValues(command string, patterns []*regexp.Regexp, display bool) (string, bool) {
	if len(patterns) == 0 {
		return command, true
	}
	type wordSpan struct{ start, end int }
	type replacement struct {
		start, end int
		value      string
	}
	words := make([]wordSpan, 0)
	for index := 0; index < len(command); {
		character, size := utf8.DecodeRuneInString(command[index:])
		if unicode.IsSpace(character) || strings.ContainsRune("|&;()<>", character) {
			index += size
			continue
		}
		start := index
		var quote rune
		escaped := false
		for index < len(command) {
			character, size = utf8.DecodeRuneInString(command[index:])
			if escaped {
				escaped = false
				index += size
				continue
			}
			if character == '\\' && quote != '\'' {
				escaped = true
				index += size
				continue
			}
			if quote != 0 {
				if character == quote {
					quote = 0
				}
				index += size
				continue
			}
			if character == '\'' || character == '"' {
				quote = character
				index += size
				continue
			}
			if unicode.IsSpace(character) || strings.ContainsRune("|&;()<>", character) {
				break
			}
			index += size
		}
		words = append(words, wordSpan{start: start, end: index})
	}

	placeholder := `""`
	if display {
		placeholder = "<redacted>"
	}
	replacements := make([]replacement, 0)
	for _, word := range words {
		raw := command[word.start:word.end]
		assignment := raw
		quotedAssignment := len(raw) >= 2 &&
			((raw[0] == '\'' && raw[len(raw)-1] == '\'') || (raw[0] == '"' && raw[len(raw)-1] == '"'))
		if quotedAssignment {
			assignment = raw[1 : len(raw)-1]
		}
		if escapedEquals := strings.Index(assignment, `\=`); escapedEquals > 0 {
			nameGroups := environmentAssignmentName.FindStringSubmatch(assignment[:escapedEquals])
			if nameGroups != nil {
				for _, pattern := range patterns {
					if pattern.MatchString(nameGroups[1]) {
						return "", false
					}
				}
			}
		}
		equals := strings.IndexByte(assignment, '=')
		if equals > 0 {
			assignmentName := assignment[:equals]
			nameCandidate := assignmentName
			indexed := false
			if bracket := strings.IndexByte(assignmentName, '['); bracket > 0 {
				nameCandidate = assignmentName[:bracket]
				indexed = true
			}
			nameGroups := environmentAssignmentName.FindStringSubmatch(nameCandidate)
			if nameGroups != nil {
				name := nameGroups[1]
				for _, pattern := range patterns {
					if pattern.MatchString(name) {
						if indexed {
							return "", false
						}
						value := assignment[equals+1:]
						if strings.Contains(value, "$") || strings.Contains(value, "`") ||
							strings.Contains(command, "<(") || strings.Contains(command, ">(") ||
							(value == "" && word.end < len(command) && command[word.end] == '(') {
							return "", false
						}
						if quotedAssignment {
							replacements = append(replacements, replacement{
								start: word.start,
								end:   word.end,
								value: assignmentName + "=" + placeholder,
							})
						} else {
							replacements = append(replacements, replacement{
								start: word.start + equals + 1,
								end:   word.end,
								value: placeholder,
							})
						}
						break
					}
				}
			}
		}
	}
	if len(replacements) == 0 {
		return command, true
	}
	var result strings.Builder
	cursor := 0
	for _, replacement := range replacements {
		result.WriteString(command[cursor:replacement.start])
		result.WriteString(replacement.value)
		cursor = replacement.end
	}
	result.WriteString(command[cursor:])
	return result.String(), true
}

// Prepare applies visibility and redaction rules. Insert is what Tab returns;
// display is the safe palette label. An empty display means it equals insert.
func (settings Settings) Prepare(command, family string) (insert, display string, visible bool) {
	family = strings.ToLower(strings.TrimSpace(family))
	if len(settings.Commands.OnlyFamilies) > 0 && !containsFold(settings.Commands.OnlyFamilies, family) {
		return "", "", false
	}
	if containsFold(settings.Commands.HiddenFamilies, family) {
		return "", "", false
	}
	words := normalizedWords(command)
	for _, prefix := range settings.Commands.HiddenPrefixes {
		if hasWordPrefix(words, normalizedWords(prefix)) {
			return "", "", false
		}
	}
	for _, pattern := range settings.hiddenRegex {
		if pattern.MatchString(command) {
			return "", "", false
		}
	}

	insert = rewriteFlagValues(command, settings.Commands.RedactFlagValues, false)
	display = rewriteFlagValues(command, settings.Commands.RedactFlagValues, true)
	var safe bool
	insert, safe = rewriteEnvironmentValues(insert, settings.sensitiveVariableRegex, false)
	if !safe {
		return "", "", false
	}
	display, safe = rewriteEnvironmentValues(display, settings.sensitiveVariableRegex, true)
	if !safe {
		return "", "", false
	}
	insert = strings.TrimSpace(insert)
	display = strings.TrimSpace(display)
	if display == insert {
		display = ""
	}
	return insert, display, insert != ""
}

func Write(path string, settings Settings, force bool) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	if path == "" {
		path = DefaultPath()
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
