package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/takymt/cflcli/internal/model"
)

// WritePagesTable writes pages as a table.
func WritePagesTable(out io.Writer, pages []model.Page, includeStatus bool) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	if includeStatus {
		if _, err := fmt.Fprintln(w, "ID\tTITLE\tSTATUS"); err != nil {
			return err
		}
		for _, page := range pages {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", page.ID, page.Title, page.Status); err != nil {
				return err
			}
		}
		return w.Flush()
	}

	if _, err := fmt.Fprintln(w, "ID\tTITLE"); err != nil {
		return err
	}
	for _, page := range pages {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", page.ID, page.Title); err != nil {
			return err
		}
	}
	return w.Flush()
}
