package page

import "strings"

type fenceStart struct {
	marker string
	info   string
}

func parseFenceStart(line string) (fenceStart, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "```"):
		return fenceStart{
			marker: "```",
			info:   strings.TrimSpace(strings.TrimPrefix(trimmed, "```")),
		}, true
	case strings.HasPrefix(trimmed, "~~~"):
		return fenceStart{
			marker: "~~~",
			info:   strings.TrimSpace(strings.TrimPrefix(trimmed, "~~~")),
		}, true
	default:
		return fenceStart{}, false
	}
}

func isFenceClose(line string, marker string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), marker)
}
