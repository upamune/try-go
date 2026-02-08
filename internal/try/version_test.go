package trypkg

import "testing"

func TestBuildVersionString(t *testing.T) {
	origVersion := version
	origCommit := commit
	origShortCommit := shortCommit
	t.Cleanup(func() {
		version = origVersion
		commit = origCommit
		shortCommit = origShortCommit
	})

	tests := []struct {
		name        string
		version     string
		commit      string
		shortCommit string
		want        string
	}{
		{
			name:        "uses short commit when provided",
			version:     "1.2.3",
			commit:      "abcdef123456",
			shortCommit: "abcdef1",
			want:        "1.2.3 (abcdef1)",
		},
		{
			name:        "falls back to first 7 chars of full commit",
			version:     "1.2.3",
			commit:      "abcdef123456",
			shortCommit: "",
			want:        "1.2.3 (abcdef1)",
		},
		{
			name:        "shows only version when commit is unavailable",
			version:     "1.2.3",
			commit:      "",
			shortCommit: "",
			want:        "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version = tt.version
			commit = tt.commit
			shortCommit = tt.shortCommit
			if got := buildVersionString(); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
