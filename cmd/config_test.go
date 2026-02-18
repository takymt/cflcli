package cmd

import (
	"bytes"
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
}

func TestConfigInit_Interactive(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	opts := &configInitOptions{
		configPath: configPath,
	}

	input := "myprofile\nmysite.atlassian.net\nme@example.com\nMYSPACE\n"
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
}

func TestConfigInit_InteractiveDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Create first profile
	opts1 := &configInitOptions{
		name:       "first",
		domain:     "shared.atlassian.net",
		user:       "user@example.com",
		spaceKey:   "FIRST",
		configPath: configPath,
	}
	if err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts1); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Create second profile interactively, accepting defaults for domain and user
	opts2 := &configInitOptions{
		configPath: configPath,
	}
	// name, domain (empty=default), user (empty=default), space_key
	input := "second\n\n\nSECOND\n"
	out := &bytes.Buffer{}
	if err := runConfigInit(strings.NewReader(input), out, opts2); err != nil {
		t.Fatalf("second init failed: %v", err)
	}

	// Verify prompts show defaults
	output := out.String()
	if !strings.Contains(output, "[shared.atlassian.net]") {
		t.Errorf("expected domain default in prompt, got: %s", output)
	}
	if !strings.Contains(output, "[user@example.com]") {
		t.Errorf("expected user default in prompt, got: %s", output)
	}

	// Verify second profile inherited defaults
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[1]
	if p.Name != "second" {
		t.Errorf("expected name %q, got %q", "second", p.Name)
	}
	if p.Domain != "shared.atlassian.net" {
		t.Errorf("expected domain %q, got %q", "shared.atlassian.net", p.Domain)
	}
	if p.User != "user@example.com" {
		t.Errorf("expected user %q, got %q", "user@example.com", p.User)
	}
	if p.SpaceKey != "SECOND" {
		t.Errorf("expected space_key %q, got %q", "SECOND", p.SpaceKey)
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
		configPath: configPath,
	}
	err := runConfigInit(strings.NewReader(""), &bytes.Buffer{}, opts2)
	if err == nil {
		t.Error("expected error for duplicate profile, got nil")
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
			{Name: "work", Domain: "work.atlassian.net", User: "work@example.com", SpaceKey: "WORK"},
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
		t.Fatal("expected error when deleting current profile, got nil")
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles unchanged, got %d", len(loaded.Profiles))
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
