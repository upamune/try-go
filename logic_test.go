package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitURI(t *testing.T) {
	cases := []string{
		"https://github.com/user/repo",
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
	}
	for _, c := range cases {
		if _, ok := parseGitURI(c); !ok {
			t.Fatalf("expected parse success: %s", c)
		}
	}
}

func TestUniqueDirName(t *testing.T) {
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "2026-02-07-test"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := uniqueDirName(base, "2026-02-07-test")
	if got != "2026-02-07-test-2" {
		t.Fatalf("got %s", got)
	}
}

func TestFuzzyScore(t *testing.T) {
	if _, ok := fuzzyScore("alpha-beta", "ab"); !ok {
		t.Fatal("expected match")
	}
	if _, ok := fuzzyScore("alpha-beta", "zz"); ok {
		t.Fatal("expected no match")
	}
}

func TestScriptRename(t *testing.T) {
	cmds := scriptRename("/tmp/tries", "old", "new")
	if len(cmds) != 4 {
		t.Fatalf("unexpected len: %d", len(cmds))
	}
	if cmds[1] != "mv 'old' 'new'" {
		t.Fatalf("unexpected mv command: %s", cmds[1])
	}
}

func TestScriptDelete(t *testing.T) {
	cmds := scriptDelete("/tmp/tries", []string{"a", "b"})
	if len(cmds) != 3 {
		t.Fatalf("unexpected len: %d", len(cmds))
	}
	if cmds[1] != "test -d 'a' && rm -rf 'a'" {
		t.Fatalf("unexpected delete command: %s", cmds[1])
	}
}
