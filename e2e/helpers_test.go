package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/takymt/cflcli/internal/config"
)

var cliBinaryPath string
var cliRepoRoot string

func TestMain(m *testing.M) {
	cliRepoRoot = findRepoRoot()

	tmpDir, err := os.MkdirTemp("", "cflcli-test-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create temp dir failed: %v\n", err)
		os.Exit(1)
	}

	cliBinaryPath = filepath.Join(tmpDir, "cfl")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", cliBinaryPath, ".")
	build.Dir = cliRepoRoot
	build.Env = os.Environ()
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		_ = os.RemoveAll(tmpDir)
		_, _ = fmt.Fprintf(os.Stderr, "build failed: %v\n%s", buildErr, string(out))
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func findRepoRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filepath.Dir(currentFile))
}

func runCLI(t *testing.T, xdgConfigHome string, stdin string, args ...string) (string, error) {
	t.Helper()
	return runCLIWithEnv(t, xdgConfigHome, stdin, nil, args...)
}

func runCLIWithEnv(t *testing.T, xdgConfigHome string, stdin string, extraEnv map[string]string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), cliBinaryPath, args...)
	cmd.Dir = cliRepoRoot
	env := withEnv(os.Environ(), "XDG_CONFIG_HOME", xdgConfigHome)
	for key, value := range extraEnv {
		env = withEnv(env, key, value)
	}
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

func withEnv(base []string, key string, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(base)+1)
	for _, item := range base {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		result = append(result, item)
	}
	return append(result, prefix+value)
}

func configPath(xdgConfigHome string) string {
	return filepath.Join(xdgConfigHome, "cflcli", "config.toml")
}

func loadConfig(t *testing.T, xdgConfigHome string) *config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(configPath(xdgConfigHome))
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	return cfg
}

func createProfile(t *testing.T, xdgConfigHome string, name string, domain string, user string, spaceKey string, output string) {
	t.Helper()
	out, err := runCLI(t, xdgConfigHome, "", "config", "init", name, "--domain", domain, "--user", user, "--space-key", spaceKey, "--profile-output", output)
	if err != nil {
		t.Fatalf("create profile %q failed: %v\n%s", name, err, out)
	}
}
