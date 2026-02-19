package cmd

import (
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

func TestValidatePageListLimit(t *testing.T) {
	testCases := []struct {
		name    string
		limit   int
		wantErr bool
	}{
		{name: "too small", limit: 0, wantErr: true},
		{name: "too large", limit: 251, wantErr: true},
		{name: "lower bound", limit: 1, wantErr: false},
		{name: "upper bound", limit: 250, wantErr: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validatePageListLimit(tc.limit)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveProfile(t *testing.T) {
	testCases := []struct {
		name        string
		profileFlag string
		cfg         *config.Config
		wantName    string
		wantContain string
	}{
		{
			name:        "uses explicitly selected profile",
			profileFlag: "personal",
			cfg: &config.Config{
				Current: "work",
				Profiles: []config.Profile{
					{Name: "work"},
					{Name: "personal"},
				},
			},
			wantName: "personal",
		},
		{
			name:        "rejects unknown selected profile",
			profileFlag: "missing",
			cfg: &config.Config{
				Current: "work",
				Profiles: []config.Profile{
					{Name: "work"},
				},
			},
			wantContain: `profile "missing" not found`,
		},
		{
			name:        "uses current profile when flag is empty",
			profileFlag: "",
			cfg: &config.Config{
				Current: "work",
				Profiles: []config.Profile{
					{Name: "work"},
					{Name: "personal"},
				},
			},
			wantName: "work",
		},
		{
			name:        "rejects when no current profile",
			profileFlag: "",
			cfg: &config.Config{
				Current:  "",
				Profiles: []config.Profile{{Name: "work"}},
			},
			wantContain: "no current profile",
		},
	}

	prev := ProfileFlag()
	defer SetProfileFlag(prev)

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			SetProfileFlag(tc.profileFlag)

			got, err := resolveProfile(tc.cfg)
			if tc.wantContain != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantContain)
				}
				if !strings.Contains(err.Error(), tc.wantContain) {
					t.Fatalf("expected error containing %q, got %v", tc.wantContain, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatalf("expected profile %q, got nil", tc.wantName)
			}
			if got.Name != tc.wantName {
				t.Fatalf("unexpected profile: got %q want %q", got.Name, tc.wantName)
			}
		})
	}
}

func TestNewPageListCmd_ArgsValidation(t *testing.T) {
	cmd := newPageListCmd()

	if err := cmd.Args(cmd, []string{"unexpected"}); err == nil {
		t.Fatalf("expected args validation error for extra positional args")
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("unexpected args validation error for no args: %v", err)
	}
}
