package trypkg

import "testing"

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
