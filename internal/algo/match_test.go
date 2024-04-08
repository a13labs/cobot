package algo_test

import (
	"testing"

	"github.com/a13labs/cobot/internal/algo"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern  string
		strings  []string
		expected []string
	}{
		{
			pattern:  "*.yaml",
			strings:  []string{"file.yaml", "file.yml", "file.json"},
			expected: []string{"file.yaml"},
		},
		{
			pattern:  "file.*",
			strings:  []string{"file.yaml", "file.yml", "file.json"},
			expected: []string{"file.yaml", "file.yml", "file.json"},
		},
		{
			pattern:  "*/file.*",
			strings:  []string{"dir/file.yaml", "dir/file.yml", "dir/file.json", "file.yaml", "file.yml", "file.json", "dir/file"},
			expected: []string{"dir/file.yaml", "dir/file.yml", "dir/file.json"},
		},
	}

	for _, test := range tests {
		matches := algo.MatchAll(test.pattern, test.strings)
		if len(matches) != len(test.expected) {
			t.Errorf("Expected %v, got %v", test.expected, matches)
		}
		for i, match := range matches {
			if match != test.expected[i] {
				t.Errorf("Expected %v, got %v", test.expected, matches)
			}
		}
	}
}
