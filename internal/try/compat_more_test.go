// Package trypkg tests compatibility behavior and command script generation.
package trypkg

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRunExecCdGitURLShorthand(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "cd https://github.com/user/repo", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "git clone 'https://github.com/user/repo'") {
		t.Fatalf("expected clone via cd shorthand: %s", joined)
	}
}

func TestRunCloneCustomNameHasDatePrefix(t *testing.T) {
	base := t.TempDir()
	script, err := runClone(base, "https://github.com/user/repo", "my app")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !regexp.MustCompile(`/\d{4}-\d{2}-\d{2}-my-app`).MatchString(joined) {
		t.Fatalf("expected date-prefix custom name: %s", joined)
	}
}

func TestParseGitURIForGenericHost(t *testing.T) {
	gt, ok := parseGitURI("https://gitlab.com/user/repo.git")
	if !ok {
		t.Fatal("expected parse success")
	}
	if gt.Host != "gitlab.com" || gt.User != "user" || gt.Repo != "repo" {
		t.Fatalf("unexpected parse result: %#v", gt)
	}
}

func TestDeleteToggleTwiceThenEscCancels(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))

	_, err := runExec(base, "cd alpha", execOptions{
		AndKeys: parseTestKeys("CTRL-D,CTRL-D,ESC"),
	})
	if err == nil {
		t.Fatal("expected cancellation")
	}
	if err != errCancelled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorktreeScriptAlwaysStartsWithMkdir(t *testing.T) {
	base := t.TempDir()
	script, err := runExec(base, "worktree feature", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(script) == 0 || !strings.HasPrefix(script[0], "mkdir -p ") {
		t.Fatalf("expected mkdir first: %#v", script)
	}
}
