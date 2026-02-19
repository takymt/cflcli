package e2e

import (
	"strings"
	"testing"
)

func TestUseCommandsSwitchCurrentProfile(t *testing.T) {
	xdgConfigHome := t.TempDir()

	createProfile(t, xdgConfigHome, "default", "default.atlassian.net", "default@example.com", "DEF", "table")
	createProfile(t, xdgConfigHome, "work", "work.atlassian.net", "work@example.com", "WORK", "table")
	createProfile(t, xdgConfigHome, "personal", "personal.atlassian.net", "personal@example.com", "HOME", "json")

	commands := []struct {
		name        string
		args        []string
		wantCurrent string
		wantOutput  string
	}{
		{
			name:        "root use command",
			args:        []string{"use", "work"},
			wantCurrent: "work",
			wantOutput:  `Switched to profile "work".`,
		},
		{
			name:        "config use alias command",
			args:        []string{"config", "use", "personal"},
			wantCurrent: "personal",
			wantOutput:  `Switched to profile "personal".`,
		},
	}

	for _, cmdCase := range commands {
		cmdCase := cmdCase
		t.Run(cmdCase.name, func(t *testing.T) {
			out, err := runCLI(t, xdgConfigHome, "", cmdCase.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v\n%s", err, out)
			}
			if !strings.Contains(out, cmdCase.wantOutput) {
				t.Fatalf("unexpected output: %s", out)
			}

			cfg := loadConfig(t, xdgConfigHome)
			if cfg.Current != cmdCase.wantCurrent {
				t.Fatalf("expected current %q, got %q", cmdCase.wantCurrent, cfg.Current)
			}
		})
	}
}
