package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takymt/cflcli/internal/config"
	"golang.org/x/term"
)

type useOptions struct {
	configPath string
}

func newUseCmd() *cobra.Command {
	opts := &useOptions{}

	cmd := &cobra.Command{
		Use:   "use [name]",
		Short: "Switch to a profile",
		Long:  "Switch to a profile by name, or interactively select one.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("too many arguments\nUsage: cfl use [name]")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runUse(cmd.OutOrStdout(), args[0], opts)
			}
			// In production, use raw terminal for interactive input
			return runUseInteractiveRaw(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

func runUse(out io.Writer, name string, opts *useOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.SetCurrent(name); err != nil {
		return err
	}

	if err := saveConfig(cfg, opts.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Switched to profile %q.\n", name)
	return nil
}

// runUseInteractive is the testable core of interactive profile selection.
// Prompts and input are done via out and in.
func runUseInteractive(in io.Reader, out io.Writer, opts *useOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		_, _ = fmt.Fprintln(out, "No profiles configured. Run 'cfl config init' to create one.")
		return nil
	}

	_, _ = fmt.Fprintln(out, "Select a profile:")
	for i, p := range cfg.Profiles {
		marker := ""
		if p.Name == cfg.Current {
			marker = " (current)"
		}
		_, _ = fmt.Fprintf(out, "  %d) %s%s\n", i+1, p.Name, marker)
	}
	_, _ = fmt.Fprint(out, "Enter number: ")

	input, cancelled, err := readInput(in)
	if err != nil {
		return err
	}
	if cancelled {
		_, _ = fmt.Fprintln(out, "selection cancelled")
		return nil
	}

	return selectProfile(out, input, cfg, opts)
}

// runUseInteractiveRaw is the production version that uses raw terminal mode.
// Prompts go to stderr so they don't pollute stdout.
func runUseInteractiveRaw(out io.Writer, opts *useOptions) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		_, _ = fmt.Fprintln(out, "No profiles configured. Run 'cfl config init' to create one.")
		return nil
	}

	_, _ = fmt.Fprintln(os.Stderr, "Select a profile:")
	for i, p := range cfg.Profiles {
		marker := ""
		if p.Name == cfg.Current {
			marker = " (current)"
		}
		_, _ = fmt.Fprintf(os.Stderr, "  %d) %s%s\n", i+1, p.Name, marker)
	}
	_, _ = fmt.Fprint(os.Stderr, "Enter number: ")

	input, cancelled, err := readLineRaw()
	if err != nil {
		return err
	}
	if cancelled {
		_, _ = fmt.Fprintln(os.Stderr, "selection cancelled")
		return nil
	}

	return selectProfile(out, input, cfg, opts)
}

func selectProfile(out io.Writer, input string, cfg *config.Config, opts *useOptions) error {
	input = strings.TrimSpace(input)

	num, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Errorf("invalid selection: %q", input)
	}

	if num < 1 || num > len(cfg.Profiles) {
		return fmt.Errorf("invalid selection: must be between 1 and %d", len(cfg.Profiles))
	}

	name := cfg.Profiles[num-1].Name
	if err := cfg.SetCurrent(name); err != nil {
		return err
	}

	if err := saveConfig(cfg, opts.configPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Switched to profile %q.\n", name)
	return nil
}

// readInput reads a line from the reader and returns the trimmed input.
// The second return value is true if the input was cancelled (EOF or ESC).
func readInput(in io.Reader) (string, bool, error) {
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", true, fmt.Errorf("read input: %w", err)
	}
	input := strings.TrimSpace(line)
	if strings.Contains(input, "\x1b") {
		return "", true, nil
	}
	return input, false, nil
}

// readLineRaw reads a line from the terminal in raw mode.
// ESC, Ctrl+C, and Ctrl+D cause immediate cancellation.
func readLineRaw() (string, bool, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", true, fmt.Errorf("set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var buf []byte
	b := make([]byte, 1)
	for {
		_, readErr := os.Stdin.Read(b)
		if readErr != nil {
			return "", true, fmt.Errorf("read input: %w", readErr)
		}
		switch b[0] {
		case 0x1b: // ESC
			_, _ = fmt.Fprint(os.Stderr, "\r\n")
			return "", true, nil
		case 0x03: // Ctrl+C
			_, _ = fmt.Fprint(os.Stderr, "\r\n")
			return "", true, nil
		case 0x04: // Ctrl+D
			_, _ = fmt.Fprint(os.Stderr, "\r\n")
			return "", true, nil
		case 0x0d: // Enter
			_, _ = fmt.Fprint(os.Stderr, "\r\n")
			return string(buf), false, nil
		case 0x7f: // Backspace
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				_, _ = fmt.Fprint(os.Stderr, "\b \b")
			}
		default:
			buf = append(buf, b[0])
			_, _ = fmt.Fprint(os.Stderr, string(b[0]))
		}
	}
}
