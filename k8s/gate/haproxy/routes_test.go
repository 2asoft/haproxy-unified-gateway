package haproxy

import (
	"testing"
)

func TestSanitizeRegexp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no dots",
			input:    "abc",
			expected: "abc",
		},
		{
			name:     "single dot",
			input:    "a.b",
			expected: "a\\.b",
		},
		{
			name:     "multiple dots",
			input:    "a.b.c",
			expected: "a\\.b\\.c",
		},
		{
			name:     "dot star",
			input:    "a.*b",
			expected: "a.*b",
		},
		{
			name:     "dot star and single dot",
			input:    "a.*b.c",
			expected: "a.*b\\.c",
		},
		{
			name:     "single dot and dot star",
			input:    "a.b.*c",
			expected: "a\\.b.*c",
		},
		{
			name:     "starts with dot",
			input:    ".abc",
			expected: "\\.abc",
		},
		{
			name:     "starts with dot star",
			input:    ".*abc",
			expected: ".*abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeRegexp(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeRegexp(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
