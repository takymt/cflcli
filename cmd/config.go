package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/config"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration profiles",
	}

	configCmd.AddCommand(newConfigInitCmd())
	configCmd.AddCommand(newConfigEditCmd())
	configCmd.AddCommand(newConfigUseCmd())
	configCmd.AddCommand(newConfigListCmd())
	configCmd.AddCommand(newConfigShowCmd())
	configCmd.AddCommand(newConfigDeleteCmd())

	return configCmd
}

type configInitOptions struct {
	name       string
	domain     string
	user       string
	spaceKey   string
	output     string
	configPath string

	defaultProfileName string
	updateExisting     bool
}

func newConfigInitCmd() *cobra.Command {
	opts := &configInitOptions{}

	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new profile interactively",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl config init [name]")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := resolveConfigInitName(opts.name, args)
			if err != nil {
				return err
			}

			runOpts := *opts
			runOpts.name = name
			return runConfigInit(cmd.InOrStdin(), cmd.OutOrStdout(), &runOpts)
		},
	}

	cmd.Flags().StringVar(&opts.name, "name", "", "profile name")
	cmd.Flags().StringVar(&opts.domain, "domain", "", "Confluence domain (e.g. mysite.atlassian.net)")
	cmd.Flags().StringVar(&opts.user, "user", "", "email address")
	cmd.Flags().StringVar(&opts.spaceKey, "space-key", "", "default space key")
	cmd.Flags().StringVar(&opts.output, "profile-output", "", "default output format for this profile (json | table)")

	return cmd
}

type configEditOptions struct {
	domain     string
	user       string
	spaceKey   string
	output     string
	configPath string
}

type configUseOptions struct {
	configPath string
}

func newConfigUseCmd() *cobra.Command {
	opts := &configUseOptions{}

	cmd := &cobra.Command{
		Use:   "use [name]",
		Short: "Switch to a profile",
		Long:  "Switch to a profile by name, or interactively select one.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl config use [name]")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runConfigUse(cmd.OutOrStdout(), args[0], opts)
			}
			return runConfigUseInteractiveRaw(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

func newConfigEditCmd() *cobra.Command {
	opts := &configEditOptions{}

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an existing profile interactively",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("profile name is required\nUsage: cfl config edit <name>")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl config edit <name>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigEdit(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.domain, "domain", "", "Confluence domain (e.g. mysite.atlassian.net)")
	cmd.Flags().StringVar(&opts.user, "user", "", "email address")
	cmd.Flags().StringVar(&opts.spaceKey, "space-key", "", "default space key")
	cmd.Flags().StringVar(&opts.output, "profile-output", "", "default output format for this profile (json | table)")

	return cmd
}

func loadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		return config.LoadFrom(configPath)
	}
	return config.Load()
}

func saveConfig(cfg *config.Config, configPath string) error {
	if configPath != "" {
		return cfg.SaveTo(configPath)
	}
	return cfg.Save()
}

func runConfigInit(in io.Reader, out io.Writer, opts *configInitOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return runConfigInitWithConfig(in, out, opts, cfg)
}

func runConfigEdit(in io.Reader, out io.Writer, name string, opts *configEditOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.FindProfile(name) == nil {
		return fmt.Errorf("profile %q not found", name)
	}

	runOpts := &configInitOptions{
		name:               name,
		domain:             opts.domain,
		user:               opts.user,
		spaceKey:           opts.spaceKey,
		output:             opts.output,
		configPath:         opts.configPath,
		defaultProfileName: name,
		updateExisting:     true,
	}
	return runConfigInitWithConfig(in, out, runOpts, cfg)
}

func runConfigUse(out io.Writer, name string, opts *configUseOptions) error {
	return runUse(out, name, &useOptions{configPath: opts.configPath})
}

func runConfigUseInteractiveRaw(out io.Writer, opts *configUseOptions) error {
	return runUseInteractiveRaw(out, &useOptions{configPath: opts.configPath})
}

func runConfigInitWithConfig(in io.Reader, out io.Writer, opts *configInitOptions, cfg *config.Config) error {
	reader := bufio.NewReader(in)
	var err error

	defaultProfile := configInitDefaults(cfg, opts.defaultProfileName)
	if defaultProfile == nil {
		defaultProfile = &config.Profile{}
	}

	name := opts.name
	if name == "" {
		name, err = prompt(reader, out, "Profile Name", "")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if !opts.updateExisting && cfg.FindProfile(name) != nil {
		return fmt.Errorf("profile %q already exists", name)
	}

	domain := opts.domain
	if domain == "" {
		domain, err = prompt(reader, out, "Domain", defaultProfile.Domain)
		if err != nil {
			return err
		}
	}
	if domain == "" {
		return fmt.Errorf("domain is required")
	}

	user := opts.user
	if user == "" {
		user, err = prompt(reader, out, "Email", defaultProfile.User)
		if err != nil {
			return err
		}
	}
	if user == "" {
		return fmt.Errorf("email address is required")
	}

	spaceKey := opts.spaceKey
	if spaceKey == "" {
		spaceKey, err = prompt(reader, out, "Space Key", defaultProfile.SpaceKey)
		if err != nil {
			return err
		}
	}
	if spaceKey == "" {
		return fmt.Errorf("space key is required")
	}

	profileOutput := opts.output
	if profileOutput == "" {
		profileOutput, err = prompt(reader, out, "Output (json|table)", defaultOutputFormat(defaultProfile.Output))
		if err != nil {
			return err
		}
	}
	profileOutput, err = normalizeOutputFormat(profileOutput)
	if err != nil {
		return err
	}

	profile := &config.Profile{
		Name:     name,
		Domain:   domain,
		User:     user,
		SpaceKey: spaceKey,
		Output:   profileOutput,
	}

	if opts.updateExisting {
		existing := cfg.FindProfile(name)
		if existing == nil {
			return fmt.Errorf("profile %q not found", name)
		}
		*existing = *profile
	} else {
		if err := cfg.AddProfile(profile); err != nil {
			return err
		}
	}

	if err := saveConfig(cfg, opts.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if opts.updateExisting {
		_, _ = fmt.Fprintf(out, "Profile %q updated successfully.\n", name)
		return nil
	}

	_, _ = fmt.Fprintf(out, "Profile %q created successfully.\n", name)
	if cfg.Current == name {
		_, _ = fmt.Fprintln(out, "Set as current profile.")
	}
	return nil
}

type configListOptions struct {
	configPath string
}

func newConfigListCmd() *cobra.Command {
	opts := &configListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigList(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

func runConfigList(out io.Writer, opts *configListOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		_, _ = fmt.Fprintln(out, "No profiles configured. Run 'cfl config init' to create one.")
		return nil
	}

	// Pass 1: format with tabwriter using plain text
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "  NAME\tDOMAIN\tSPACE")
	for _, p := range cfg.Profiles {
		marker := " "
		if p.Name == cfg.Current {
			marker = "*"
		}
		_, _ = fmt.Fprintf(w, "%s %s\t%s\t%s\n", marker, p.Name, p.Domain, p.SpaceKey)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// Pass 2: colorize current profile line
	green := color.New(color.FgGreen)
	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "*") {
			_, _ = green.Fprintln(out, line)
		} else {
			_, _ = fmt.Fprintln(out, line)
		}
	}
	return nil
}

type configShowOptions struct {
	configPath string
}

func newConfigShowCmd() *cobra.Command {
	opts := &configShowOptions{}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current profile details",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShow(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

func runConfigShow(out io.Writer, opts *configShowOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	p := cfg.CurrentProfile()
	if p == nil {
		_, _ = fmt.Fprintln(out, "No current profile. Run 'cfl config init' to create one.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintf(w, "Name:\t%s\n", p.Name)
	_, _ = fmt.Fprintf(w, "Domain:\t%s\n", p.Domain)
	_, _ = fmt.Fprintf(w, "User:\t%s\n", p.User)
	_, _ = fmt.Fprintf(w, "Space Key:\t%s\n", p.SpaceKey)
	_, _ = fmt.Fprintf(w, "Output:\t%s\n", outputForDisplay(p.Output))
	return w.Flush()
}

type configDeleteOptions struct {
	configPath string
	force      bool
}

func newConfigDeleteCmd() *cobra.Command {
	opts := &configDeleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("profile name is required\nUsage: cfl config delete <name>")
			}
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl config delete <name>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigDelete(cmd.OutOrStdout(), args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.force, "force", false, "allow deleting current profile")

	return cmd
}

func runConfigDelete(out io.Writer, name string, opts *configDeleteOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	switchedToDefault := false
	if cfg.Current == name && !opts.force {
		return fmt.Errorf("cannot delete current profile %q without --force", name)
	}
	if cfg.Current == name && opts.force {
		if name == "default" {
			return fmt.Errorf("cannot delete current profile %q with --force", name)
		}
		if cfg.FindProfile("default") == nil {
			return fmt.Errorf("cannot delete current profile %q with --force: profile %q not found", name, "default")
		}
		cfg.Current = "default"
		switchedToDefault = true
	}

	if err := cfg.DeleteProfile(name); err != nil {
		return err
	}

	if err := saveConfig(cfg, opts.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Profile %q deleted.\n", name)
	if switchedToDefault {
		_, _ = fmt.Fprintln(out, `Current profile switched to "default".`)
	}
	return nil
}

func prompt(reader *bufio.Reader, out io.Writer, label string, defaultVal string) (string, error) {
	if defaultVal != "" {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, defaultVal)
	} else {
		_, _ = fmt.Fprintf(out, "%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	val := strings.TrimSpace(line)
	if val == "" {
		return defaultVal, nil
	}
	return val, nil
}

func configInitDefaults(cfg *config.Config, profileName string) *config.Profile {
	source := strings.TrimSpace(profileName)
	if source == "" {
		source = "default"
	}
	return cfg.FindProfile(source)
}

func defaultOutputFormat(value string) string {
	output, err := normalizeOutputFormat(value)
	if err != nil {
		return "table"
	}
	return output
}

func resolveConfigInitName(flagName string, args []string) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("too many arguments\nUsage: cfl config init [name]")
	}
	if len(args) == 0 {
		return strings.TrimSpace(flagName), nil
	}

	argName := strings.TrimSpace(args[0])
	if flagName != "" && argName != flagName {
		return "", fmt.Errorf("argument %q conflicts with --name %q", argName, flagName)
	}
	return argName, nil
}

func normalizeOutputFormat(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "json", "table":
		return value, nil
	default:
		return "", fmt.Errorf("output must be one of: json, table")
	}
}

func outputForDisplay(value string) string {
	if output, err := normalizeOutputFormat(value); err == nil {
		return output
	}
	return "table"
}
