package cli

import "testing"

func TestIsWatchQuitByte(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   byte
		want bool
	}{
		{in: 'q', want: true},
		{in: 'Q', want: true},
		{in: 3, want: true}, // Ctrl+C in raw mode
		{in: 'x', want: false},
		{in: '\n', want: false},
	}

	for _, tt := range tests {
		if got := isWatchQuitByte(tt.in); got != tt.want {
			t.Fatalf("isWatchQuitByte(%v) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
