package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestRunExecCloneSubcommand(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "clone https://github.com/user/repo", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "git clone 'https://github.com/user/repo'") {
		t.Fatalf("missing clone command: %s", joined)
	}
	if !regexp.MustCompile(`/\d{4}-\d{2}-\d{2}-user-repo`).MatchString(joined) {
		t.Fatalf("missing date-prefixed path: %s", joined)
	}
}

func TestRunExecURLShorthandClone(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "https://github.com/user/repo", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "git clone 'https://github.com/user/repo'") {
		t.Fatalf("missing clone command: %s", joined)
	}
}

func TestRunExecWorktreeSubcommand(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "worktree feature-branch", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "worktree add --detach") {
		t.Fatalf("missing worktree command: %s", joined)
	}
	if !strings.Contains(joined, "feature-branch") {
		t.Fatalf("missing worktree name: %s", joined)
	}
}

func TestRunExecDotRequiresName(t *testing.T) {
	base := t.TempDir()
	_, err := runExec(base, ".", execOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires a name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExecDotWithNameUsesWorktreeWhenGitExists(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, ". myfeature", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "worktree add --detach") {
		t.Fatalf("expected worktree command: %s", joined)
	}
}

func TestRunExecAndKeysCreateNew(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "cd", execOptions{AndKeys: parseTestKeys("TYPE=alpha,ENTER")})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "mkdir -p") {
		t.Fatalf("expected mkdir script: %s", joined)
	}
	if !strings.Contains(joined, "-alpha") {
		t.Fatalf("expected alpha suffix: %s", joined)
	}
}

func TestInitScriptEmbedsPath(t *testing.T) {
	s := initScript("/tmp/tries")
	if !strings.Contains(s, "exec --path '/tmp/tries'") {
		t.Fatalf("missing path embedding: %s", s)
	}
	if !strings.Contains(s, "eval") {
		t.Fatalf("missing eval behavior: %s", s)
	}
}

func TestRunExecAndExitRenderOnly(t *testing.T) {
	base := t.TempDir()
	_, err := runExec(base, "cd alpha", execOptions{AndExit: true})
	if err == nil {
		t.Fatal("expected render-only sentinel")
	}
	if err != errRenderOnly {
		t.Fatalf("unexpected error: %v", err)
	}
}
