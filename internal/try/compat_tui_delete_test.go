package trypkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExecEscCancels(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))
	_, err := runExec(base, "cd", execOptions{AndKeys: parseTestKeys("ESC")})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if err != errCancelled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExecEnterSelectsFilteredDir(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))
	mustMkdir(t, filepath.Join(base, "beta"))

	script, err := runExec(base, "cd alpha", execOptions{AndKeys: parseTestKeys("ENTER")})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, filepath.Join(base, "alpha")) {
		t.Fatalf("expected alpha selection: %s", joined)
	}
}

func TestRunExecDeleteFlow(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))

	script, err := runExec(base, "cd alpha", execOptions{
		AndKeys:    parseTestKeys("CTRL-D,ENTER,ENTER"),
		AndConfirm: "YES",
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "rm -rf 'alpha'") {
		t.Fatalf("expected delete script: %s", joined)
	}
	if !strings.Contains(joined, "cd '"+base+"'") {
		t.Fatalf("expected base cd: %s", joined)
	}
}

func TestRunExecDeleteCancelledWhenNotYes(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))

	_, err := runExec(base, "cd alpha", execOptions{
		AndKeys:    parseTestKeys("CTRL-D,ENTER,ENTER"),
		AndConfirm: "NO",
	})
	if err == nil {
		t.Fatal("expected cancellation")
	}
	if err != errCancelled {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExecRenameFlow(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "alpha"))

	script, err := runExec(base, "cd alpha", execOptions{
		AndKeys: parseTestKeys("CTRL-R,CTRL-A,CTRL-K,TYPE=renamed,ENTER"),
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "mv 'alpha' 'renamed'") {
		t.Fatalf("expected rename script: %s", joined)
	}
}

func TestRunExecCtrlWEditing(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "beta"))

	script, err := runExec(base, "cd", execOptions{
		AndKeys: parseTestKeys("TYPE=hello-world,CTRL-W,TYPE=beta,ENTER"),
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "-hello-beta") {
		t.Fatalf("expected ctrl-w word delete behavior: %s", joined)
	}
}

func TestRunExecDotPathWithoutGitFallsBackToMkdir(t *testing.T) {
	base := t.TempDir()
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	script, err := runExec(base, ". custom", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if strings.Contains(joined, "worktree add --detach") {
		t.Fatalf("unexpected worktree command: %s", joined)
	}
	if !strings.Contains(joined, "mkdir -p") {
		t.Fatalf("expected mkdir script: %s", joined)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
