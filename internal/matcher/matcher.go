package matcher

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/PrashikshitSaini/Deja/internal/model"
)

var simpleWrappers = map[string]bool{
	"builtin":   true,
	"command":   true,
	"exec":      true,
	"nocorrect": true,
	"noglob":    true,
	"time":      true,
}

var sudoOptionsWithValues = map[string]bool{
	"-C": true, "-D": true, "-g": true, "-h": true, "-p": true,
	"-R": true, "-r": true, "-T": true, "-u": true,
	"--chdir": true, "--chroot": true, "--close-from": true,
	"--group": true, "--host": true, "--prompt": true,
	"--role": true, "--type": true, "--user": true,
}

// Words performs best-effort shell tokenization and tolerates partial input.
func Words(input string) []string {
	var words []string
	var current strings.Builder
	var quote rune
	escaped := false
	started := false
	flush := func() {
		if started {
			words = append(words, current.String())
			current.Reset()
			started = false
		}
	}

	for _, character := range input {
		if escaped {
			current.WriteRune(character)
			started = true
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			started = true
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
				started = true
			}
			continue
		}
		if character == '\'' || character == '"' {
			started = true
			quote = character
			continue
		}
		if unicode.IsSpace(character) {
			flush()
			continue
		}
		if strings.ContainsRune("|&;()", character) {
			flush()
			words = append(words, string(character))
			continue
		}
		current.WriteRune(character)
		started = true
	}
	if escaped {
		current.WriteRune('\\')
	}
	flush()
	return words
}

func assignment(token string) bool {
	equals := strings.IndexByte(token, '=')
	if equals < 1 {
		return false
	}
	for index, character := range token[:equals] {
		if index == 0 && !(character == '_' || unicode.IsLetter(character)) {
			return false
		}
		if index > 0 && !(character == '_' || unicode.IsLetter(character) || unicode.IsDigit(character)) {
			return false
		}
	}
	return true
}

func skipOptions(tokens []string, index int, withValues map[string]bool) int {
	for index < len(tokens) && strings.HasPrefix(tokens[index], "-") {
		option := tokens[index]
		index++
		if withValues[option] && index < len(tokens) {
			index++
		}
	}
	return index
}

// Family returns the executable or shell builtin that owns a command line.
func Family(command string) string {
	tokens := Words(command)
	for index := 0; index < len(tokens); {
		token := tokens[index]
		if assignment(token) {
			index++
			continue
		}

		name := strings.ToLower(filepath.Base(strings.TrimPrefix(token, "\\")))
		if simpleWrappers[name] {
			index++
			index = skipOptions(tokens, index, nil)
			continue
		}
		if name == "env" {
			index++
			index = skipOptions(tokens, index, map[string]bool{
				"-u": true, "--unset": true, "-C": true, "--chdir": true,
				"-S": true, "--split-string": true,
			})
			for index < len(tokens) && assignment(tokens[index]) {
				index++
			}
			continue
		}
		if name == "sudo" {
			index++
			index = skipOptions(tokens, index, sudoOptionsWithValues)
			for index < len(tokens) && assignment(tokens[index]) {
				index++
			}
			continue
		}
		if strings.Contains("|&;()", token) {
			return ""
		}
		return name
	}
	return ""
}

func tokensMatch(query, command string) bool {
	queryText := strings.ToLower(strings.TrimSpace(query))
	if queryText == "" {
		return false
	}
	commandText := strings.ToLower(strings.TrimSpace(command))
	if strings.HasPrefix(commandText, queryText) || strings.Contains(commandText, queryText) {
		return true
	}

	queryWords := Words(queryText)
	commandWords := Words(commandText)
	commandIndex := 0
	for _, queryWord := range queryWords {
		matched := false
		for commandIndex < len(commandWords) {
			candidateWord := commandWords[commandIndex]
			commandIndex++
			if strings.HasPrefix(candidateWord, queryWord) || strings.Contains(candidateWord, queryWord) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return len(queryWords) > 0
}

// Rank filters and orders variants using only explainable, local signals.
func Rank(candidates []model.Candidate, query, cwd string, limit int, now float64) []model.Candidate {
	queryText := strings.ToLower(strings.TrimSpace(query))
	requestedFamily := Family(query)
	ranked := make([]model.Candidate, 0, len(candidates))

	for _, candidate := range candidates {
		displayCommand := candidate.DisplayCommand()
		if !tokensMatch(query, displayCommand) {
			continue
		}
		commandText := strings.ToLower(strings.TrimSpace(displayCommand))
		score := 0.0
		if requestedFamily == candidate.Family {
			score += 60
		} else if strings.HasPrefix(candidate.Family, requestedFamily) {
			score += 25
		}
		if commandText == queryText {
			score += 130
		} else if strings.HasPrefix(commandText, queryText) {
			score += 100
		} else if strings.Contains(commandText, queryText) {
			score += 50
		}
		score += math.Min(30, math.Log2(float64(candidate.Uses+1))*7)
		if cwd != "" && candidate.CWDHits > 0 {
			score += math.Min(35, 15+math.Log2(float64(candidate.CWDHits+1))*5)
		}
		if successRate, ok := candidate.SuccessRate(); ok {
			score += successRate * 10
		}
		if candidate.LastRun > 0 {
			ageDays := math.Max(0, now-candidate.LastRun) / 86_400
			score += 20 / (1 + ageDays/14)
		}
		candidate.Score = score
		ranked = append(ranked, candidate)
	}

	sort.SliceStable(ranked, func(left, right int) bool {
		if ranked[left].Score != ranked[right].Score {
			return ranked[left].Score > ranked[right].Score
		}
		if ranked[left].Uses != ranked[right].Uses {
			return ranked[left].Uses > ranked[right].Uses
		}
		if ranked[left].LastRun != ranked[right].LastRun {
			return ranked[left].LastRun > ranked[right].LastRun
		}
		return ranked[left].Command < ranked[right].Command
	})
	if limit < 0 {
		limit = 0
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}
