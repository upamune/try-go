package main

import "testing"

func TestNormalizeArgsWithPathOnly(t *testing.T) {
	got := normalizeArgs([]string{"--path", "/tmp/tries"})
	want := []string{"--path", "/tmp/tries", "exec"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestNormalizeArgsKeepsCommand(t *testing.T) {
	got := normalizeArgs([]string{"--path=/tmp/tries", "clone", "https://github.com/u/r"})
	want := []string{"--path=/tmp/tries", "clone", "https://github.com/u/r"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestParseTestKeysTokenMode(t *testing.T) {
	got := parseTestKeys("CTRL-D,ENTER,Y,E,S,ENTER")
	if len(got) != 6 {
		t.Fatalf("unexpected keys len: %#v", got)
	}
	if got[0] != "\x04" || got[1] != "\r" || got[2] != "Y" {
		t.Fatalf("unexpected mapping: %#v", got)
	}
}
