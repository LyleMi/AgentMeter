package query

import (
	"sort"
	"time"
)

const maxAnalyticsGapSpanDays = 62

func fillAnalyticsDateGaps[T any](items []T, dateOf func(T) string, empty func(string) T) []T {
	if len(items) <= 1 {
		return items
	}
	sorted := append([]T(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return dateOf(sorted[i]) < dateOf(sorted[j]) })
	start, end, ok := analyticsDateRange(dateOf(sorted[0]), dateOf(sorted[len(sorted)-1]))
	if !ok {
		return sorted
	}
	spanDays := int(end.Sub(start).Hours()/24) + 1
	if spanDays <= len(sorted) || spanDays > maxAnalyticsGapSpanDays {
		return sorted
	}
	byDate := make(map[string]T, len(sorted))
	for _, item := range sorted {
		byDate[dateOf(item)] = item
	}
	filled := make([]T, 0, spanDays)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format(analyticsDateOnlyLayout)
		item, ok := byDate[date]
		if !ok {
			item = empty(date)
		}
		filled = append(filled, item)
	}
	return filled
}

func analyticsDateRange(first, last string) (time.Time, time.Time, bool) {
	start, startErr := time.Parse(analyticsDateOnlyLayout, first)
	end, endErr := time.Parse(analyticsDateOnlyLayout, last)
	return start, end, startErr == nil && endErr == nil && !end.Before(start)
}
