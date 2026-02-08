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

	cmd := exec.Command(
		"script",
		"-q",
		"/dev/null",
		"zsh",
		"-lc",
		"out=$(go run ./cmd/try exec --and-type foo --and-exit 2>/dev/tty); :",
	)
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
