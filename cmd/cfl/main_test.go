package main

import "testing"

func TestNormalizeConfluenceDomain(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{
			name:  "plain domain",
			input: "example.atlassian.net",
			want:  "example.atlassian.net",
		},
		{
			name:  "https prefixed domain",
			input: "https://example.atlassian.net",
			want:  "example.atlassian.net",
		},
		{
			name:      "empty",
			input:     "",
			wantError: true,
		},
		{
			name:      "wrong host",
			input:     "example.com",
			wantError: true,
		},
		{
			name:      "contains path",
			input:     "example.atlassian.net/wiki",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeConfluenceDomain(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("normalizeConfluenceDomain() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeConfluenceDomain() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeConfluenceDomain() = %q, want %q", got, tt.want)
			}
		})
	}
}
