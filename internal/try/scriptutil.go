package trypkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `"'"'`) + "'"
}

func shellDoubleQuote(s string) string {
	s = strings.ReplaceAll(s, `\\`, `\\\\`)
	s = strings.ReplaceAll(s, `"`, `\\"`)
	s = strings.ReplaceAll(s, "$", `\\$`)
	return s
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
	cmd := fmt.Sprintf("/usr/bin/env sh -c 'if git -C \"%s\" rev-parse --is-inside-work-tree >/dev/null 2>&1; then root=$(git -C \"%s\" rev-parse --show-toplevel); git -C \"$root\" worktree add --detach \"%s\" >/dev/null 2>&1 || true; fi; exit 0'",
		shellDoubleQuote(repo), shellDoubleQuote(repo), shellDoubleQuote(path))
	return append([]string{
		"mkdir -p " + shellQuote(path),
		"echo " + shellQuote("Using git worktree to create this trial from "+repo+"."),
		cmd,
	}, scriptCD(path)...)
}

func scriptDelete(names []string, basePath string) []string {
	cmds := []string{"cd " + shellQuote(basePath)}
	for _, name := range names {
		cmds = append(cmds, "test -d "+shellQuote(name)+" && rm -rf "+shellQuote(name))
	}
	cmds = append(cmds, "( cd "+shellQuote(os.Getenv("PWD"))+" 2>/dev/null || cd \"$HOME\" )")
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
  if out=$(%s exec%s "$@" 2>/dev/tty); then
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
