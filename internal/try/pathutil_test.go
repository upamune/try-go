package trypkg

import (
	"os"
	"path/filepath"
	"testing"
)

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
