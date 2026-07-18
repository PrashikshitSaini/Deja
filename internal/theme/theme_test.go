package theme

import "testing"

func TestRankSymbolsUseGeometricMarkers(t *testing.T) {
	tests := map[string]string{
		"dark-red": "🔴",
		"red":      "🔻",
		"orange":   "🔶",
		"gold":     "⭐",
		"default":  "·",
	}
	for color, want := range tests {
		if got := Symbol(color); got != want {
			t.Errorf("Symbol(%q) = %q, want %q", color, got, want)
		}
	}
}
