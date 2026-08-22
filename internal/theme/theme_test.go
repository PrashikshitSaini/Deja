package theme

import "testing"

func TestANSIColorsValidateAndResolve(t *testing.T) {
	if !Valid("dim") || Valid("neon") {
		t.Fatal("Valid disagrees with the color table")
	}
	if ANSI("dim") != "\x1b[2m" {
		t.Fatalf("ANSI(dim) = %q", ANSI("dim"))
	}
}
