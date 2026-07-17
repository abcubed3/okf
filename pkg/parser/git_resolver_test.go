package parser

import "testing"

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"https://github.com/abcubed3/okf.git", true},
		{"http://github.com/abcubed3/okf.git", true},
		{"git@github.com:abcubed3/okf.git", true},
		{"ssh://git@github.com/abcubed3/okf.git", true},
		{"git://github.com/abcubed3/okf.git", true},
		{"my-bundle-v1.0.tar.gz", false},
		{"/Users/abcubed3/okf-go", false},
		{"./local/path/to/bundle", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsGitURL(tt.path); got != tt.want {
				t.Errorf("IsGitURL(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
