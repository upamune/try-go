package trypkg

import (
	"os"
	"testing"
	"time"
)

func TestFuzzyScore(t *testing.T) {
	if _, _, ok := fuzzyScore("alpha-beta", "ab"); !ok {
		t.Fatal("expected match")
	}
	if _, _, ok := fuzzyScore("alpha-beta", "zz"); ok {
		t.Fatal("expected no match")
	}
}

func TestFuzzyScorePositions(t *testing.T) {
	_, pos, ok := fuzzyScore("alpha-beta", "ab")
	if !ok {
		t.Fatal("expected match")
	}
	if len(pos) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(pos))
	}
	// 'a' at index 0, 'b' at index 6
	if pos[0] != 0 {
		t.Fatalf("expected first position 0, got %d", pos[0])
	}
	if pos[1] != 6 {
		t.Fatalf("expected second position 6, got %d", pos[1])
	}
}

func TestFuzzyScoreEmptyQuery(t *testing.T) {
	score, pos, ok := fuzzyScore("anything", "")
	if !ok {
		t.Fatal("expected match for empty query")
	}
	if score != 0 {
		t.Fatalf("expected score 0 for empty query, got %f", score)
	}
	if pos != nil {
		t.Fatalf("expected nil positions for empty query, got %v", pos)
	}
}

func TestFilterEntriesShorterNameWins(t *testing.T) {
	now := time.Now()
	entries := []dirEntry{
		{Name: "2024-01-01-abcdef", MTime: now, Base: 2.0},
		{Name: "2024-01-01-abc", MTime: now, Base: 2.0},
	}
	results := filterEntries(entries, "abc")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Shorter name should rank first due to length penalty
	if results[0].Entry.Name != "2024-01-01-abc" {
		t.Fatalf("expected shorter name first, got %s", results[0].Entry.Name)
	}
}

func TestFilterEntriesDensityBoost(t *testing.T) {
	now := time.Now()
	entries := []dirEntry{
		{Name: "a------b------c", MTime: now, Base: 0},
		{Name: "abc", MTime: now, Base: 0},
	}
	results := filterEntries(entries, "abc")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Dense match "abc" should score higher
	if results[0].Entry.Name != "abc" {
		t.Fatalf("expected dense match first, got %s", results[0].Entry.Name)
	}
}

func TestRenderNameHighlighted(t *testing.T) {
	// Disable colors for predictable output
	setNoColors(true)
	defer setNoColors(false)

	result := renderNameHighlighted("alpha-beta", []int{0, 6}, 20)
	// With no-color bold style, should contain the original characters
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	// Test truncation
	result = renderNameHighlighted("very-long-name-here", []int{0}, 8)
	runes := []rune(result)
	// Should end with ellipsis
	if runes[len(runes)-1] != '…' {
		t.Fatalf("expected truncation ellipsis, got %q", result)
	}
}

func TestRenderNameHighlightedDatePrefix(t *testing.T) {
	setNoColors(true)
	defer setNoColors(false)

	// Date-prefixed name should not crash
	result := renderNameHighlighted("2024-01-15-foo", []int{11, 12, 13}, 30)
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestEnvIntOr(t *testing.T) {
	key := "TEST_ENV_INT_OR_12345"

	// Unset: fallback
	os.Unsetenv(key)
	if v := envIntOr(key, 42); v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}

	// Valid int
	os.Setenv(key, "100")
	if v := envIntOr(key, 42); v != 100 {
		t.Fatalf("expected 100, got %d", v)
	}

	// Invalid: fallback
	os.Setenv(key, "notanumber")
	if v := envIntOr(key, 42); v != 42 {
		t.Fatalf("expected 42 for invalid, got %d", v)
	}

	os.Unsetenv(key)
}
