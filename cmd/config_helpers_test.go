package cmd

import "testing"

func TestNormalizeOutputFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "json", input: "json", want: "json"},
		{name: "table", input: "table", want: "table"},
		{name: "trim and lower", input: "  JSON  ", want: "json"},
		{name: "invalid", input: "yaml", wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeOutputFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeOutputFormat: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestOutputForDisplay(t *testing.T) {
	t.Parallel()

	if got := outputForDisplay("json"); got != "json" {
		t.Fatalf("got %q want %q", got, "json")
	}
	if got := outputForDisplay("bogus"); got != "table" {
		t.Fatalf("got %q want %q", got, "table")
	}
}

func TestResolveConfigInitName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		flagName string
		args     []string
		want     string
		wantErr  bool
	}{
		{name: "from flag", flagName: "work", want: "work"},
		{name: "from arg", args: []string{"work"}, want: "work"},
		{name: "same arg and flag", flagName: "work", args: []string{"work"}, want: "work"},
		{name: "conflict", flagName: "work", args: []string{"dev"}, wantErr: true},
		{name: "too many args", args: []string{"a", "b"}, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveConfigInitName(tc.flagName, tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveConfigInitName: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
