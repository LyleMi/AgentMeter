package query

import (
	"strconv"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

const analyticsDateOnlyLayout = "2006-01-02"

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
		args = append(args, normalizeAnalyticsDateBoundary(filters.StartedFrom, "start"))
	}
	if strings.TrimSpace(filters.StartedTo) != "" {
		toValue, exclusive := normalizeAnalyticsToBoundary(filters.StartedTo)
		if exclusive {
			where = append(where, startedExpr+" < ?")
		} else {
			where = append(where, startedExpr+" <= ?")
		}
		args = append(args, toValue)
	}
	return where, args
}

func normalizeAnalyticsDateBoundary(value, boundary string) string {
	value = strings.TrimSpace(value)
	if date, ok := parseAnalyticsDateOnly(value); ok {
		if boundary == "end" {
			date = date.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return date.Format(time.RFC3339Nano)
	}
	return value
}

func normalizeAnalyticsToBoundary(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if date, ok := parseAnalyticsDateOnly(value); ok {
		return date.AddDate(0, 0, 1).Format(time.RFC3339Nano), true
	}
	return value, false
}

func parseAnalyticsDateOnly(value string) (time.Time, bool) {
	date, err := time.ParseInLocation(analyticsDateOnlyLayout, strings.TrimSpace(value), time.UTC)
	return date, err == nil
}
