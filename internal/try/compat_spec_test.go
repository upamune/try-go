package trypkg

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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

func TestRunExecWorktreeDirSubcommand(t *testing.T) {
	base := t.TempDir()
	repoDir := t.TempDir()
	script, err := runExec(base, "worktree "+repoDir+" mybranch", execOptions{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "worktree add --detach") {
		t.Fatalf("missing worktree command: %s", joined)
	}
	if !strings.Contains(joined, repoDir) {
		t.Fatalf("missing repo dir in script: %s", joined)
	}
	if !strings.Contains(joined, "mybranch") {
		t.Fatalf("missing worktree name: %s", joined)
	}
}

func TestRunWorktreeWithExplicitDir(t *testing.T) {
	base := t.TempDir()
	repoDir := t.TempDir()
	script, err := runWorktree(base, repoDir, "feat")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(script, "\n")
	if !strings.Contains(joined, "worktree add --detach") {
		t.Fatalf("missing worktree command: %s", joined)
	}
	if !strings.Contains(joined, repoDir) {
		t.Fatalf("expected repo dir in script: %s", joined)
	}
}

func TestResolveWorktreeArgs(t *testing.T) {
	// Both args given
	rd, wn := resolveWorktreeArgs("/some/dir", "name")
	if rd != "/some/dir" || wn != "name" {
		t.Fatalf("expected (/some/dir, name), got (%s, %s)", rd, wn)
	}

	// No args
	rd, wn = resolveWorktreeArgs("", "")
	if rd != "" || wn != "" {
		t.Fatalf("expected empty, got (%s, %s)", rd, wn)
	}

	// Single arg that is a directory
	tmpDir := t.TempDir()
	rd, wn = resolveWorktreeArgs(tmpDir, "")
	if rd != tmpDir || wn != "" {
		t.Fatalf("expected dir=%s name=empty, got (%s, %s)", tmpDir, rd, wn)
	}

	// Single arg that is not a directory → treat as name
	rd, wn = resolveWorktreeArgs("not-a-dir-xyz", "")
	if rd != "" || wn != "not-a-dir-xyz" {
		t.Fatalf("expected dir=empty name=not-a-dir-xyz, got (%s, %s)", rd, wn)
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
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	script, err := runExec(base, "./repo myfeature", execOptions{})
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

func TestExecEmitsANSIWhenStdoutIsCapturedAndStderrIsTTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("script-based tty test is not supported on windows")
	}
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script command is required")
	}
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is required")
	}

	var cmd *exec.Cmd
	shellCmd := "out=$(go run ./cmd/try exec --and-type foo --and-exit 2>/dev/tty); :"
	if runtime.GOOS == "linux" {
		cmd = exec.Command("script", "-q", "-c", "zsh -lc '"+shellCmd+"'", "/dev/null")
	} else {
		cmd = exec.Command("script", "-q", "/dev/null", "zsh", "-lc", shellCmd)
	}
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "NO_COLOR=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script command failed: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("\x1b[")) {
		t.Fatalf("expected ANSI color/style escape sequences, got:\n%s", out)
	}
}
