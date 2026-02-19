package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/config"
)

type initOptions struct {
	configPath string
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize cfl configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.InOrStdin(), cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

func runInit(in io.Reader, out io.Writer, opts *initOptions) error {
	configPath, err := getConfigPath(opts.configPath)
	if err != nil {
		return err
	}

	initialized, err := isInitialized(configPath)
	if err != nil {
		return err
	}
	if initialized {
		_, _ = fmt.Fprintln(out, "already initialized; no changes made.")
		return nil
	}

	if err := runConfigInit(in, out, &configInitOptions{
		name:       "default",
		configPath: opts.configPath,
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Initialization completed.")
	return nil
}

func getConfigPath(configPath string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return config.Path()
}

func isInitialized(configPath string) (bool, error) {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat config: %w", err)
	}
	return true, nil
}
