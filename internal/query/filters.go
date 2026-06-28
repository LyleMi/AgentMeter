package query

import (
	"strconv"
	"strings"
)

func appendSourceFilter(where []string, args []any, value string) ([]string, []any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return where, args
	}
	if idText, ok := strings.CutPrefix(value, "source:"); ok {
		id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
		if err != nil || id <= 0 {
			return append(where, "src.id = ?"), append(args, int64(-1))
		}
		return append(where, "src.id = ?"), append(args, id)
	}
	return append(where, "src.kind = ?"), append(args, value)
}
