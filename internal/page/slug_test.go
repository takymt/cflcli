package page

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	t.Parallel()

	got, err := GenerateSlug()
	if err != nil {
		t.Fatalf("GenerateSlug() error = %v", err)
	}
	if len(got) != 16 {
		t.Fatalf("GenerateSlug() len = %d, want 16", len(got))
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Fatalf("GenerateSlug() = %q, want lowercase hex: %v", got, err)
	}
}

func TestValidateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		slug      string
		wantError bool
	}{
		{
			name:      "empty",
			slug:      "",
			wantError: true,
		},
		{
			name: "single character",
			slug: "a",
		},
		{
			name: "max length",
			slug: strings.Repeat("a", maxSlugLength),
		},
		{
			name:      "too long",
			slug:      strings.Repeat("a", maxSlugLength+1),
			wantError: true,
		},
		{
			name: "allows free form",
			slug: "Team Plan v2.final",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSlug(tt.slug)
			if tt.wantError && err == nil {
				t.Fatal("ValidateSlug() error = nil, want non-nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("ValidateSlug() error = %v, want nil", err)
			}
		})
	}
}
