package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadTTYLine_CRAndLF(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"y\n", "y"},
		{"y\r", "y"},
		{"yes\r\n", "yes"},
		{"", ""},
	}
	for _, tc := range tests {
		got, err := ReadTTYLine(strings.NewReader(tc.in))
		if err != nil && tc.in != "" {
			t.Fatalf("ReadTTYLine(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ReadTTYLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestReadTTYLine_EOF(t *testing.T) {
	t.Parallel()
	got, err := ReadTTYLine(bytes.NewReader(nil))
	if err == nil {
		t.Fatalf("expected EOF, got line %q", got)
	}
}
