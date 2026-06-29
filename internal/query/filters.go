package query

import (
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
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
	where, args = appendProjectFilter(where, args, filters.Project, "s.project_path")
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

func appendProjectFilter(where []string, args []any, value, expr string) ([]string, []any) {
	key := projectFilterKey(value)
	if key == "" {
		return where, args
	}
	return append(where, normalizedProjectPathSQL(expr)+" = ?"), append(args, key)
}

func normalizedProjectPathSQL(expr string) string {
	normalized := "RTRIM(REPLACE(COALESCE(" + expr + ", ''), '\\', '/'), '/.')"
	if runtime.GOOS == "windows" {
		return "LOWER(" + normalized + ")"
	}
	return normalized
}

func projectFilterKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	normalized := strings.ReplaceAll(sourcepath.Normalize(value), "\\", "/")
	normalized = strings.TrimRight(normalized, "/.")
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	return normalized
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
