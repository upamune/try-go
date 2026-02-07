package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	httpsGitRE = regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+?)(?:\.git)?$`)
	sshGitRE   = regexp.MustCompile(`^git@([^:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
)

type gitTarget struct {
	Host string
	User string
	Repo string
}

func parseGitURI(raw string) (gitTarget, bool) {
	raw = strings.TrimSpace(raw)
	if m := httpsGitRE.FindStringSubmatch(raw); m != nil {
		return gitTarget{Host: m[1], User: m[2], Repo: m[3]}, true
	}
	if m := sshGitRE.FindStringSubmatch(raw); m != nil {
		return gitTarget{Host: m[1], User: m[2], Repo: m[3]}, true
	}
	return gitTarget{}, false
}

func isGitURI(raw string) bool {
	_, ok := parseGitURI(raw)
	return ok
}

func cloneDirName(basePath, rawURL, custom string) (string, error) {
	if strings.TrimSpace(custom) != "" {
		name := sanitizeName(custom)
		if name == "" {
			return "", fmt.Errorf("invalid custom name")
		}
		today := time.Now().Format("2006-01-02")
		return uniqueDirName(basePath, fmt.Sprintf("%s-%s", today, name)), nil
	}
	gt, ok := parseGitURI(rawURL)
	if !ok {
		return "", fmt.Errorf("unable to parse git URI: %s", rawURL)
	}
	today := time.Now().Format("2006-01-02")
	return uniqueDirName(basePath, fmt.Sprintf("%s-%s-%s", today, gt.User, gt.Repo)), nil
}

func uniqueDirName(basePath, dir string) string {
	candidate := dir
	for i := 2; ; i++ {
		_, err := os.Stat(filepath.Join(basePath, candidate))
		if os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", dir, i)
	}
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.Join(strings.Fields(s), "-")
	return s
}

func expandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~/"))
	}
	return filepath.Abs(p)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `"'"'`) + "'"
}

func scriptCD(path string) []string {
	return []string{
		"touch " + shellQuote(path),
		"echo " + shellQuote(path),
		"cd " + shellQuote(path),
	}
}

func scriptMkdirCD(path string) []string {
	return append([]string{"mkdir -p " + shellQuote(path)}, scriptCD(path)...)
}

func scriptClone(path, uri string) []string {
	return append([]string{
		"mkdir -p " + shellQuote(path),
		"echo " + shellQuote("Using git clone to create this trial from "+uri+"."),
		"git clone " + shellQuote(uri) + " " + shellQuote(path),
	}, scriptCD(path)...)
}

func scriptWorktree(path, repo string) []string {
	cmd := fmt.Sprintf("/usr/bin/env sh -c 'if git -C %s rev-parse --is-inside-work-tree >/dev/null 2>&1; then root=$(git -C %s rev-parse --show-toplevel); git -C \"$root\" worktree add --detach %s >/dev/null 2>&1 || true; fi; exit 0'",
		shellQuote(repo), shellQuote(repo), shellQuote(path))
	return append([]string{
		"mkdir -p " + shellQuote(path),
		"echo " + shellQuote("Using git worktree to create this trial from "+repo+"."),
		cmd,
	}, scriptCD(path)...)
}

func scriptDelete(basePath string, names []string) []string {
	cmds := []string{"cd " + shellQuote(basePath)}
	for _, name := range names {
		cmds = append(cmds, "test -d "+shellQuote(name)+" && rm -rf "+shellQuote(name))
	}
	return cmds
}

func scriptRename(basePath, oldName, newName string) []string {
	newPath := filepath.Join(basePath, newName)
	return []string{
		"cd " + shellQuote(basePath),
		"mv " + shellQuote(oldName) + " " + shellQuote(newName),
		"echo " + shellQuote(newPath),
		"cd " + shellQuote(newPath),
	}
}

func initScript(basePath string) string {
	exe, err := os.Executable()
	if err != nil {
		exe = "try"
	}
	exe = strings.ReplaceAll(exe, "'", "")
	pathArg := " --path " + shellQuote(basePath)
	bash := fmt.Sprintf(`try() {
  local out
  out=$(%s exec%s "$@" 2>/dev/tty)
  if [ $? -eq 0 ]; then
    eval "$out"
  else
    echo "$out"
  fi
}`,
		shellQuote(exe), pathArg,
	)
	fish := fmt.Sprintf(`function try
  set -l out (%s exec%s $argv 2>/dev/tty | string collect)
  if test $pipestatus[1] -eq 0
    eval $out
  else
    echo $out
  end
end`, shellQuote(exe), pathArg)

	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "fish") {
		return fish
	}
	return bash
}
