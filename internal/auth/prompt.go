package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type Prompter interface {
	Prompt(label string) (string, error)
	PromptSecret(label string) (string, error)
}

type TerminalPrompter struct {
	stdin  *os.File
	stdout io.Writer
}

func NewTerminalPrompter(stdin *os.File, stdout io.Writer) *TerminalPrompter {
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = io.Discard
	}
	return &TerminalPrompter{
		stdin:  stdin,
		stdout: stdout,
	}
}

func (p *TerminalPrompter) Prompt(label string) (string, error) {
	if _, err := fmt.Fprint(p.stdout, label+": "); err != nil {
		return "", err
	}

	line, err := bufio.NewReader(p.stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *TerminalPrompter) PromptSecret(label string) (string, error) {
	info, err := p.stdin.Stat()
	if err != nil {
		return "", err
	}
	if (info.Mode() & os.ModeCharDevice) == 0 {
		return "", errors.New("interactive api token prompt requires a terminal; use --api-token")
	}

	if _, err := fmt.Fprint(p.stdout, label+": "); err != nil {
		return "", err
	}
	raw, err := term.ReadPassword(int(p.stdin.Fd()))
	_, _ = fmt.Fprintln(p.stdout)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}
