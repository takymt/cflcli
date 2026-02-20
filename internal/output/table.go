package output

import (
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/takymt/cflcli/internal/model"
)

const pageTableGap = 3

// WritePagesTable writes pages as a table.
func WritePagesTable(out io.Writer, pages []model.Page, includeStatus bool) error {
	var rows [][]string
	if includeStatus {
		rows = append(rows, []string{"ID", "TITLE", "STATUS"})
		for _, page := range pages {
			rows = append(rows, []string{page.ID, page.Title, page.Status})
		}
	} else {
		rows = append(rows, []string{"ID", "TITLE"})
		for _, page := range pages {
			rows = append(rows, []string{page.ID, page.Title})
		}
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if w := stringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	for rowIndex, row := range rows {
		for i, cell := range row {
			b.WriteString(cell)
			if i == len(row)-1 {
				continue
			}
			pad := widths[i] - stringWidth(cell) + pageTableGap
			if pad < pageTableGap {
				pad = pageTableGap
			}
			b.WriteString(strings.Repeat(" ", pad))
		}
		b.WriteByte('\n')

		if rowIndex == 0 {
			for i, width := range widths {
				if width < 2 {
					width = 2
				}
				b.WriteString(strings.Repeat("-", width))
				if i == len(widths)-1 {
					continue
				}
				b.WriteString(strings.Repeat(" ", pageTableGap))
			}
			b.WriteByte('\n')
		}
	}

	if _, err := io.WriteString(out, b.String()); err != nil {
		return err
	}
	return nil
}

func stringWidth(s string) int {
	return runewidth.StringWidth(s)
}
