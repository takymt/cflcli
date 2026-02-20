package body

import (
	"strings"
	"testing"
)

func TestNormalizeFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "markdown", input: "markdown", want: "markdown"},
		{name: "storage", input: "storage", want: "storage"},
		{name: "trim and lower", input: "  MARKDOWN  ", want: "markdown"},
		{name: "invalid", input: "wiki", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeFormat: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestToStorage(t *testing.T) {
	t.Parallel()

	t.Run("markdown", func(t *testing.T) {
		got, err := ToStorage([]byte("hello **world**"), "markdown")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if !strings.Contains(got, "<p>") || !strings.Contains(got, "<strong>world</strong>") {
			t.Fatalf("unexpected converted value: %q", got)
		}
	})

	t.Run("storage passthrough", func(t *testing.T) {
		got, err := ToStorage([]byte("<p>Hello</p>"), "storage")
		if err != nil {
			t.Fatalf("ToStorage: %v", err)
		}
		if got != "<p>Hello</p>" {
			t.Fatalf("got %q want %q", got, "<p>Hello</p>")
		}
	})
}
