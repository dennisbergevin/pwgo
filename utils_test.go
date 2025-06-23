package main

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPlural(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
		{-1, "s"},
	}

	for _, test := range tests {
		result := plural(test.input)
		if result != test.expected {
			t.Errorf("plural(%d) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestTagStyleFor_ConsistentHashColor(t *testing.T) {
	tag := "e2e"
	style1 := tagStyleFor(tag)
	style2 := tagStyleFor(tag)

	if style1.String() != style2.String() {
		t.Errorf("Expected consistent style output for same tag, got different styles")
	}
}

func TestTagStyleFor_BackgroundMatchesHash(t *testing.T) {
	tag := "alpha"
	hash := sha256.Sum256([]byte(tag))
	r, g, b := hash[0], hash[1], hash[2]
	expectedBg := fmt.Sprintf("#%02x%02x%02x", r, g, b)

	style := tagStyleFor(tag)
	if got := style.GetBackground(); got != lipgloss.Color(expectedBg) {
		t.Errorf("Background color mismatch: got %q, want %q", got, expectedBg)
	}
}

func TestTagStyleFor_ForegroundContrast(t *testing.T) {
	tag := "dark"
	hash := sha256.Sum256([]byte(tag))
	r, g, b := hash[0], hash[1], hash[2]
	brightness := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)

	style := tagStyleFor(tag)
	fg := style.GetForeground()

	if brightness < 128 && fg != lipgloss.Color("#ffffff") {
		t.Errorf("Expected white foreground for dark background, got %s", fg)
	}
	if brightness >= 128 && fg != lipgloss.Color("#000000") {
		t.Errorf("Expected black foreground for bright background, got %s", fg)
	}
}
