package main

import "testing"

func TestFuzzyScore(t *testing.T) {
	if _, ok := fuzzyScore("alpha-beta", "ab"); !ok {
		t.Fatal("expected match")
	}
	if _, ok := fuzzyScore("alpha-beta", "zz"); ok {
		t.Fatal("expected no match")
	}
}
