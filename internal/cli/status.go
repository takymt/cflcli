package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type syncProgress interface {
	Set(msg string)
	Clear()
	Println(msg string)
}

type progressLine struct {
	writer  io.Writer
	enabled bool
	active  bool
	lastLen int
}

func newProgressLine(writer io.Writer) *progressLine {
	return &progressLine{
		writer:  writer,
		enabled: isTerminalWriter(writer),
	}
}

func (p *progressLine) Set(msg string) {
	if !p.enabled {
		return
	}
	if msg == "" {
		return
	}

	padding := ""
	if p.active && len(msg) < p.lastLen {
		padding = strings.Repeat(" ", p.lastLen-len(msg))
	}
	_, _ = fmt.Fprint(p.writer, "\r"+msg+padding)
	p.active = true
	p.lastLen = len(msg)
}

func (p *progressLine) Clear() {
	if !p.enabled || !p.active {
		return
	}
	_, _ = fmt.Fprint(p.writer, "\r"+strings.Repeat(" ", p.lastLen)+"\r")
	p.active = false
	p.lastLen = 0
}

func (p *progressLine) Println(msg string) {
	if !p.enabled {
		_, _ = fmt.Fprintln(p.writer, msg)
		return
	}

	padding := ""
	if p.active && len(msg) < p.lastLen {
		padding = strings.Repeat(" ", p.lastLen-len(msg))
	}

	if p.active {
		_, _ = fmt.Fprint(p.writer, "\r"+msg+padding+"\n")
	} else {
		_, _ = fmt.Fprintln(p.writer, msg)
	}
	p.active = false
	p.lastLen = 0
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
