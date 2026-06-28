package query

import (
	"strconv"
	"strings"

	"AgentMeter/internal/model"
)

func appendSourceFilter(where []string, args []any, value string) ([]string, []any) {
	return appendSourceFilterWithAlias(where, args, value, "src")
}

func appendSourceFilterWithAlias(where []string, args []any, value, alias string) ([]string, []any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return where, args
	}
	if idText, ok := strings.CutPrefix(value, "source:"); ok {
		id, err := strconv.ParseInt(strings.TrimSpace(idText), 10, 64)
		if err != nil || id <= 0 {
			return append(where, alias+".id = ?"), append(args, int64(-1))
		}
		return append(where, alias+".id = ?"), append(args, id)
	}
	return append(where, alias+".kind = ?"), append(args, value)
}

func appendAnalyticsFilters(where []string, args []any, filters model.AnalyticsFilters, sourceAlias, modelExpr, startedExpr string) ([]string, []any) {
	where, args = appendSourceFilterWithAlias(where, args, filters.Agent, sourceAlias)
	if strings.TrimSpace(filters.Model) != "" {
		where = append(where, modelExpr+" = ?")
		args = append(args, strings.TrimSpace(filters.Model))
	}
	if strings.TrimSpace(filters.StartedFrom) != "" {
		where = append(where, startedExpr+" >= ?")
		args = append(args, strings.TrimSpace(filters.StartedFrom))
	}
	if strings.TrimSpace(filters.StartedTo) != "" {
		where = append(where, startedExpr+" <= ?")
		args = append(args, strings.TrimSpace(filters.StartedTo))
	}
	return where, args
}
