package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Formatter renders data to an output stream.
type Formatter interface {
	Print(v any) error
}

// New creates a formatter by name.
func New(name string, out io.Writer) (Formatter, error) {
	switch name {
	case "json":
		return &jsonFormatter{out: out}, nil
	case "table":
		return &tableFormatter{out: out}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", name)
	}
}

type jsonFormatter struct {
	out io.Writer
}

func (f *jsonFormatter) Print(v any) error {
	enc := json.NewEncoder(f.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
