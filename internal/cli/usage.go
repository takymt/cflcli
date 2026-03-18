package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type usageError struct {
	cmd *cobra.Command
	err error
}

type silentError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

func (e *silentError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *silentError) Unwrap() error {
	return e.err
}

func (e *silentError) Silent() bool {
	return true
}

func newUsageError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	return &usageError{
		cmd: cmd,
		err: err,
	}
}

func newSilentError(err error) error {
	if err == nil {
		return nil
	}
	return &silentError{err: err}
}

func exactArgsWithUsage(n int) cobra.PositionalArgs {
	return positionalArgsWithUsage(cobra.ExactArgs(n))
}

func noArgsWithUsage() cobra.PositionalArgs {
	return positionalArgsWithUsage(cobra.NoArgs)
}

func positionalArgsWithUsage(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			return newUsageError(cmd, err)
		}
		return nil
	}
}

func requireFlagsWithUsage(flags ...string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		var missing []string
		for _, name := range flags {
			if !cmd.Flags().Changed(name) {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			return nil
		}
		return newUsageError(cmd, fmt.Errorf(`required flag(s) "%s" not set`, strings.Join(missing, `", "`)))
	}
}
