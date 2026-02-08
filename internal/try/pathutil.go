package trypkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
