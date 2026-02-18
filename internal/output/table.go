package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/takymt/cflcli/internal/model"
)

type tableFormatter struct {
	out io.Writer
}

func (f *tableFormatter) Print(v any) error {
	switch data := v.(type) {
	case []model.Page:
		return f.printPages(data)
	case PageListOutput:
		return f.printPages(data.Results)
	case *PageListOutput:
		return f.printPages(data.Results)
	default:
		return fmt.Errorf("unsupported table data type: %T", v)
	}
}

func (f *tableFormatter) printPages(pages []model.Page) error {
	w := tabwriter.NewWriter(f.out, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tSPACE"); err != nil {
		return err
	}
	for _, page := range pages {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", page.ID, page.Title, page.Status, page.Space.ID); err != nil {
			return err
		}
	}
	return w.Flush()
}
