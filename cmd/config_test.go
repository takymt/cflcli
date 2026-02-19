package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/takymt/cflcli/internal/config"
)

func init() {
	// Disable color output in tests for predictable assertions
	color.NoColor = true
}

func TestConfigInit_WithFlags(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		name:       "test",
		domain:     "test.atlassian.net",
		user:       "test@example.com",
		spaceKey:   "TEST",
		output:     "json",
		configPath: configPath,
	}

	out := &bytes.Buffer{}
	in := strings.NewReader("")
	if err := runConfigInit(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, `Profile "test" created successfully.`) {
		t.Errorf("unexpected output: %s", output)
	}
	if !strings.Contains(output, "Set as current profile.") {
		t.Errorf("expected 'Set as current profile.' in output: %s", output)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Current != "test" {
		t.Errorf("expected current %q, got %q", "test", cfg.Current)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if p.Name != "test" {
		t.Errorf("expected name %q, got %q", "test", p.Name)
	}
	if p.Domain != "test.atlassian.net" {
		t.Errorf("expected domain %q, got %q", "test.atlassian.net", p.Domain)
	}
	if p.User != "test@example.com" {
		t.Errorf("expected user %q, got %q", "test@example.com", p.User)
	}
	if p.SpaceKey != "TEST" {
		t.Errorf("expected space_key %q, got %q", "TEST", p.SpaceKey)
	}
	if p.Output != "json" {
		t.Errorf("expected output %q, got %q", "json", p.Output)
	}
}

func TestConfigInit_Interactive(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		configPath: configPath,
	}

	input := "myprofile\nmysite.atlassian.net\nme@example.com\nMYSPACE\njson\n"
	out := &bytes.Buffer{}
	in := strings.NewReader(input)

	if err := runConfigInit(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, `Profile "myprofile" created successfully.`) {
		t.Errorf("unexpected output: %s", output)
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if p.Name != "myprofile" {
		t.Errorf("expected name %q, got %q", "myprofile", p.Name)
	}
	if p.Domain != "mysite.atlassian.net" {
		t.Errorf("expected domain %q, got %q", "mysite.atlassian.net", p.Domain)
	}
	if p.Output != "json" {
		t.Errorf("expected output %q, got %q", "json", p.Output)
	}
}

func TestConfigInit_InteractivePromptLabels(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{configPath: configPath}

	input := "first\nnewsite.atlassian.net\nme@example.com\nFIRST\ntable\n"
	out := &bytes.Buffer{}
	in := strings.NewReader(input)

	if err := runConfigInit(in, out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Profile Name: ") {
		t.Errorf("expected profile name prompt, got: %s", output)
	}
	if !strings.Contains(output, "Domain: ") {
		t.Errorf("expected domain prompt, got: %s", output)
	}
	if !strings.Contains(output, "Email: ") {
		t.Errorf("expected email prompt, got: %s", output)
	}
	if !strings.Contains(output, "Space Key: ") {
		t.Errorf("expected space key prompt, got: %s", output)
	}
	if !strings.Contains(output, "Output (json|table) [table]: ") {
		t.Errorf("expected output prompt, got: %s", output)
	}
}

func TestConfigInit_Interactive_FirstProfileDomainIsRequired(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{configPath: configPath}

	input := "first\n\nme@example.com\nFIRST\n"
	out := &bytes.Buffer{}
	in := strings.NewReader(input)

	err := runConfigInit(in, out, opts)
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
	if !strings.Contains(err.Error(), "domain is required") {
		t.Fatalf("expected domain required error, got: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Domain: ") {
		t.Errorf("expected domain prompt, got: %s", output)
	}
}

func TestConfigInit_InteractiveDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:     "default",
				Domain:   "default.atlassian.net",
				User:     "default@example.com",
				SpaceKey: "DEF",
				Output:   "json",
			},
			{
				Name:     "work",
				Domain:   "work.atlassian.net",
				User:     "work@example.com",
				SpaceKey: "WORK",
				Output:   "table",
			},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	// Create second profile interactively, accepting all defaults from default profile.
	opts2 := &configInitOptions{
		configPath: configPath,
	}
	// name, domain, user, space_key, output
	input := "second\n\n\n\n\n"
	out := &bytes.Buffer{}
	if err := runConfigInit(strings.NewReader(input), out, opts2); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	// Verify prompts show defaults
	output := out.String()
	if !strings.Contains(output, "[default.atlassian.net]") {
		t.Errorf("expected domain default in prompt, got: %s", output)
	}
	if !strings.Contains(output, "[default@example.com]") {
		t.Errorf("expected user default in prompt, got: %s", output)
	}
	if !strings.Contains(output, "[DEF]") {
		t.Errorf("expected space key default in prompt, got: %s", output)
	}
	if !strings.Contains(output, "[json]") {
		t.Errorf("expected output default in prompt, got: %s", output)
	}

	// Verify second profile inherited defaults from "default" profile.
	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(loaded.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(loaded.Profiles))
	}
	p := loaded.Profiles[2]
	if p.Name != "second" {
		t.Errorf("expected name %q, got %q", "second", p.Name)
	}
	if p.Domain != "default.atlassian.net" {
		t.Errorf("expected domain %q, got %q", "default.atlassian.net", p.Domain)
	}
	if p.User != "default@example.com" {
		t.Errorf("expected user %q, got %q", "default@example.com", p.User)
	}
	if p.SpaceKey != "DEF" {
		t.Errorf("expected space_key %q, got %q", "DEF", p.SpaceKey)
	}
	if p.Output != "json" {
		t.Errorf("expected output %q, got %q", "json", p.Output)
	}
}

func TestConfigInit_DefaultsWithoutDefaultProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:     "work",
				Domain:   "work.atlassian.net",
				User:     "work@example.com",
				SpaceKey: "WORK",
				Output:   "json",
			},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	opts := &configInitOptions{
		configPath: configPath,
	}
	// name, domain, user, space_key, output
	input := "other\nother.atlassian.net\nother@example.com\nOTHER\n\n"
	out := &bytes.Buffer{}
	if err := runConfigInit(strings.NewReader(input), out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promptOutput := out.String()
	if strings.Contains(promptOutput, "[work.atlassian.net]") {
		t.Fatalf("did not expect current profile domain default: %s", promptOutput)
	}
	if !strings.Contains(promptOutput, "Output (json|table) [table]: ") {
		t.Fatalf("expected table default output prompt: %s", promptOutput)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	p := loaded.FindProfile("other")
	if p == nil {
		t.Fatal("expected profile 'other' to be created")
	}
	if p.Output != "table" {
		t.Fatalf("expected output %q, got %q", "table", p.Output)
	}
}

func TestConfigInit_DuplicateProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		name:       "dup",
		domain:     "a.atlassian.net",
		user:       "a@a.com",
		spaceKey:   "A",
		output:     "table",
		configPath: configPath,
	}

	if err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	opts2 := &configInitOptions{
		name:       "dup",
		domain:     "b.atlassian.net",
		user:       "b@b.com",
		spaceKey:   "B",
		output:     "table",
		configPath: configPath,
	}
	err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts2)
	if err == nil {
		t.Error("expected error for duplicate profile, got nil")
	}
}

func TestConfigInit_DuplicateProfileNameStopsEarly(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "default",
		Profiles: []config.Profile{
			{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF", Output: "table"},
			{Name: "dup", Domain: "dup.atlassian.net", User: "dup@example.com", SpaceKey: "DUP", Output: "json"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configInitOptions{configPath: configPath}
	out := &bytes.Buffer{}
	err := runConfigInit(strings.NewReader("dup\n"), out, opts)
	if err == nil {
		t.Fatal("expected duplicate profile error, got nil")
	}
	if !strings.Contains(err.Error(), `profile "dup" already exists`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "Domain: ") {
		t.Fatalf("expected to stop before domain prompt, got: %s", out.String())
	}
}

func TestConfigInitPositionalName_DuplicateProfileStopsEarly(t *testing.T) {
	xdgConfigHome := t.TempDir()

	prevXDG := os.Getenv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_CONFIG_HOME", xdgConfigHome); err != nil {
		t.Fatalf("set env failed: %v", err)
	}
	defer func() {
		if prevXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
			return
		}
		_ = os.Setenv("XDG_CONFIG_HOME", prevXDG)
	}()

	cfgPath := filepath.Join(xdgConfigHome, "cflcli", "config.toml")
	cfg := &config.Config{
		Current: "default",
		Profiles: []config.Profile{
			{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF", Output: "table"},
			{Name: "dup", Domain: "dup.atlassian.net", User: "dup@example.com", SpaceKey: "DUP", Output: "json"},
		},
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	root := NewRootCmd()
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetIn(strings.NewReader("will-not-be-used\n"))
	root.SetArgs([]string{"config", "init", "dup"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected duplicate profile error, got nil")
	}
	if !strings.Contains(err.Error(), `profile "dup" already exists`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.String(), "Domain: ") {
		t.Fatalf("expected to stop before domain prompt, got: %s", out.String())
	}
}

func TestConfigInit_InvalidOutput(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		name:       "work",
		domain:     "work.atlassian.net",
		user:       "work@example.com",
		spaceKey:   "WORK",
		output:     "yaml",
		configPath: configPath,
	}

	err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts)
	if err == nil {
		t.Fatal("expected error for invalid output, got nil")
	}
	if !strings.Contains(err.Error(), "output must be one of") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigInit_WithPositionalName(t *testing.T) {
	xdgConfigHome := t.TempDir()

	prevXDG := os.Getenv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_CONFIG_HOME", xdgConfigHome); err != nil {
		t.Fatalf("set env failed: %v", err)
	}
	defer func() {
		if prevXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
			return
		}
		_ = os.Setenv("XDG_CONFIG_HOME", prevXDG)
	}()

	root := NewRootCmd()
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetIn(strings.NewReader("site.atlassian.net\nuser@example.com\nDEV\njson\n"))
	root.SetArgs([]string{"config", "init", "work"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfgPath := filepath.Join(xdgConfigHome, "cflcli", "config.toml")
	cfg, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	p := cfg.FindProfile("work")
	if p == nil {
		t.Fatal("expected profile 'work' to be created")
	}
	if strings.Contains(out.String(), "Profile Name: ") {
		t.Fatalf("did not expect profile name prompt for positional name: %s", out.String())
	}
}

func TestConfigInit_WithPositionalNameAndFlagConflict(t *testing.T) {
	cmd := newConfigInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"arg-name", "--name", "flag-name"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "conflicts with --name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigInit_MissingName(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		configPath: configPath,
	}

	err := runConfigInit(strings.NewReader("\n"), &bytes.Buffer{}, opts)
	if err == nil {
		t.Error("expected error for missing name, got nil")
	}
}

func TestConfigInit_EOF(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{configPath: configPath}

	err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts)
	if err == nil {
		t.Error("expected error for EOF, got nil")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("expected read input error, got: %v", err)
	}
}

func TestConfigList_Empty(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configListOptions{
		configPath: configPath,
	}

	out := &bytes.Buffer{}
	if err := runConfigList(out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No profiles configured") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestConfigList_MultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configListOptions{
		configPath: configPath,
	}

	out := &bytes.Buffer{}
	if err := runConfigList(out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	if !strings.Contains(output, "NAME") {
		t.Errorf("expected header with NAME, got: %s", output)
	}

	if !strings.Contains(output, "* work") {
		t.Errorf("expected current marker on 'work', got: %s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "personal") {
			if strings.HasPrefix(strings.TrimSpace(line), "*") {
				t.Errorf("expected no marker on 'personal', got: %s", line)
			}
		}
	}

	if !strings.Contains(output, "work.atlassian.net") {
		t.Errorf("expected work domain in output: %s", output)
	}
	if !strings.Contains(output, "personal.atlassian.net") {
		t.Errorf("expected personal domain in output: %s", output)
	}
}

func TestConfigShow_WithProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK", Output: "json"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configShowOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runConfigShow(out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "work") {
		t.Errorf("expected name in output: %s", output)
	}
	if !strings.Contains(output, "work.atlassian.net") {
		t.Errorf("expected domain in output: %s", output)
	}
	if !strings.Contains(output, "work@example.com") {
		t.Errorf("expected user in output: %s", output)
	}
	if !strings.Contains(output, "WORK") {
		t.Errorf("expected space key in output: %s", output)
	}
	if !strings.Contains(output, "json") {
		t.Errorf("expected output in output: %s", output)
	}
}

func TestConfigShow_NoProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configShowOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runConfigShow(out, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No current profile") {
		t.Errorf("expected no profile message, got: %s", output)
	}
}

func TestConfigDelete_Success(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configDeleteOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runConfigDelete(out, "personal", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, `Profile "personal" deleted.`) {
		t.Errorf("unexpected output: %s", output)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "work" {
		t.Errorf("expected remaining profile %q, got %q", "work", loaded.Profiles[0].Name)
	}
}

func TestConfigDelete_CurrentProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configDeleteOptions{configPath: configPath}
	out := &bytes.Buffer{}
	err := runConfigDelete(out, "work", opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot delete current profile") {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles remaining, got %d", len(loaded.Profiles))
	}
	if loaded.Current != "work" {
		t.Errorf("expected current %q, got %q", "work", loaded.Current)
	}
}

func TestConfigDelete_CurrentProfileWithForce(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF"},
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configDeleteOptions{configPath: configPath, force: true}
	out := &bytes.Buffer{}
	if err := runConfigDelete(out, "work", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `Current profile switched to "default".`) {
		t.Fatalf("expected current switch message, got: %s", out.String())
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles remaining, got %d", len(loaded.Profiles))
	}
	if loaded.FindProfile("default") == nil {
		t.Fatal("expected default profile to remain")
	}
	if loaded.FindProfile("personal") == nil {
		t.Fatal("expected personal profile to remain")
	}
	if loaded.Current != "default" {
		t.Errorf("expected current %q, got %q", "default", loaded.Current)
	}
}

func TestConfigDelete_CurrentProfileWithForce_WithoutDefault(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configDeleteOptions{configPath: configPath, force: true}
	err := runConfigDelete(&bytes.Buffer{}, "work", opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `profile "default" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigEdit_Profile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{
				Name:     "default",
				Domain:   "default.atlassian.net",
				User:     "default@example.com",
				SpaceKey: "DEF",
				Output:   "json",
			},
			{
				Name:     "work",
				Domain:   "work.atlassian.net",
				User:     "work@example.com",
				SpaceKey: "WORK",
				Output:   "table",
			},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("save config failed: %v", err)
	}

	opts := &configEditOptions{configPath: configPath}
	in := strings.NewReader("\nnew@example.com\n\njson\n")
	out := &bytes.Buffer{}
	if err := runConfigEdit(in, out, "work", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	edited := loaded.FindProfile("work")
	if edited == nil {
		t.Fatal("expected profile 'work' to exist")
	}
	if edited.Domain != "work.atlassian.net" {
		t.Fatalf("expected domain to stay unchanged, got %q", edited.Domain)
	}
	if edited.User != "new@example.com" {
		t.Fatalf("expected user %q, got %q", "new@example.com", edited.User)
	}
	if edited.SpaceKey != "WORK" {
		t.Fatalf("expected space key %q, got %q", "WORK", edited.SpaceKey)
	}
	if edited.Output != "json" {
		t.Fatalf("expected output %q, got %q", "json", edited.Output)
	}

	if !strings.Contains(out.String(), `Profile "work" updated successfully.`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestConfigEdit_ProfileNotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configEditOptions{configPath: configPath}
	err := runConfigEdit(strings.NewReader(""), &bytes.Buffer{}, "work", opts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `profile "work" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigUse_WithName(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF"},
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configUseOptions{configPath: configPath}
	out := &bytes.Buffer{}
	if err := runConfigUse(out, "personal", opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `Switched to profile "personal".`) {
		t.Fatalf("unexpected output: %s", out.String())
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Current != "personal" {
		t.Fatalf("expected current %q, got %q", "personal", loaded.Current)
	}
}

func TestConfigUse_CommandAlias(t *testing.T) {
	xdgConfigHome := t.TempDir()

	prevXDG := os.Getenv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_CONFIG_HOME", xdgConfigHome); err != nil {
		t.Fatalf("set env failed: %v", err)
	}
	defer func() {
		if prevXDG == "" {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
			return
		}
		_ = os.Setenv("XDG_CONFIG_HOME", prevXDG)
	}()

	cfgPath := filepath.Join(xdgConfigHome, "cflcli", "config.toml")
	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "default", Domain: "default.atlassian.net", User: "default@example.com", SpaceKey: "DEF"},
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
			{Name: "personal", Domain: "personal.atlassian.net", User: "me@example.com", SpaceKey: "HOME"},
		},
	}
	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	root := NewRootCmd()
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"config", "use", "personal"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `Switched to profile "personal".`) {
		t.Fatalf("unexpected output: %s", out.String())
	}

	loaded, err := config.LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if loaded.Current != "personal" {
		t.Fatalf("expected current %q, got %q", "personal", loaded.Current)
	}
}

func TestConfigDelete_NotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &config.Config{
		Current: "work",
		Profiles: []config.Profile{
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
		},
	}
	if err := cfg.SaveTo(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	opts := &configDeleteOptions{configPath: configPath}
	out := &bytes.Buffer{}
	err := runConfigDelete(out, "nonexistent", opts)
	if err == nil {
		t.Fatal("expected error for nonexistent profile, got nil")
	}
}

func TestConfigInit_MultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	out1 := &bytes.Buffer{}
	opts1 := &configInitOptions{
		name:       "first",
		domain:     "a.atlassian.net",
		user:       "a@a.com",
		spaceKey:   "A",
		output:     "table",
		configPath: configPath,
	}
	if err := runConfigInit(strings.NewReader(""), out1, opts1); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if !strings.Contains(out1.String(), "Set as current profile.") {
		t.Errorf("first profile should be set as current")
	}

	out2 := &bytes.Buffer{}
	opts2 := &configInitOptions{
		name:       "second",
		domain:     "b.atlassian.net",
		user:       "b@b.com",
		spaceKey:   "B",
		output:     "json",
		configPath: configPath,
	}
	if err := runConfigInit(strings.NewReader(""), out2, opts2); err != nil {
		t.Fatalf("second init failed: %v", err)
	}
	if strings.Contains(out2.String(), "Set as current profile.") {
		t.Errorf("second profile should not be set as current")
	}

	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}
	if cfg.Current != "first" {
		t.Errorf("expected current %q, got %q", "first", cfg.Current)
	}
}
