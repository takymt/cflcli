package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/takymt/cflcli/internal/model"
)

// WritePagesTable writes pages as a table.
func WritePagesTable(out io.Writer, pages []model.Page) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
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
