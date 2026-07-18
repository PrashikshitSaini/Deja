package theme

import "sort"

var ansiColors = map[string]string{
	"inherit":       "",
	"default":       "\x1b[39m",
	"black":         "\x1b[30m",
	"dark-red":      "\x1b[38;5;88m",
	"red":           "\x1b[38;5;160m",
	"bright-red":    "\x1b[91m",
	"orange":        "\x1b[38;5;166m",
	"gold":          "\x1b[38;5;178m",
	"yellow":        "\x1b[33m",
	"bright-yellow": "\x1b[93m",
	"green":         "\x1b[32m",
	"cyan":          "\x1b[36m",
	"blue":          "\x1b[34m",
	"magenta":       "\x1b[35m",
	"white":         "\x1b[37m",
	"gray":          "\x1b[38;5;245m",
	"dim":           "\x1b[2m",
	"bold":          "\x1b[1m",
}

func Valid(name string) bool {
	_, ok := ansiColors[name]
	return ok
}

func ANSI(name string) string {
	return ansiColors[name]
}

// Symbol provides color that ZLE can render safely in its message area. ZLE
// escapes raw terminal control bytes, so the live palette uses colored Unicode
// markers while direct CLI output can use ANSI.
func Symbol(name string) string {
	switch name {
	case "dark-red":
		return "🔴"
	case "red", "bright-red":
		return "🔻"
	case "orange":
		return "🔶"
	case "gold", "yellow", "bright-yellow":
		return "⭐"
	case "green":
		return "✅"
	case "cyan", "blue":
		return "🔵"
	case "magenta":
		return "🔷"
	case "black", "gray":
		return "⚫"
	case "white":
		return "⚪"
	default:
		return "·"
	}
}

func Names() []string {
	names := make([]string, 0, len(ansiColors))
	for name := range ansiColors {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
