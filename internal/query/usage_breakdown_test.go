package query

import (
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestSortUsageBreakdownBuckets(t *testing.T) {
	tests := []struct {
		name    string
		groupBy string
		items   []model.UsageBreakdownBucket
		key     func(model.UsageBreakdownBucket) string
		want    []string
	}{
		{
			name: "day", groupBy: "day",
			items: []model.UsageBreakdownBucket{{Date: "2026-01-02", TotalTokens: 1}, {Date: "2026-01-01", TotalTokens: 1}},
			key:   func(item model.UsageBreakdownBucket) string { return item.Date }, want: []string{"2026-01-01", "2026-01-02"},
		},
		{
			name: "agent tie breaker", groupBy: "agent",
			items: []model.UsageBreakdownBucket{{SourceID: 2, SourceLabel: "B", SessionCount: 2}, {SourceID: 1, SourceLabel: "A", SessionCount: 2}},
			key:   func(item model.UsageBreakdownBucket) string { return item.SourceLabel }, want: []string{"A", "B"},
		},
		{
			name: "model tokens", groupBy: "model",
			items: []model.UsageBreakdownBucket{{Model: "small", TotalTokens: 1}, {Model: "large", TotalTokens: 2}},
			key:   func(item model.UsageBreakdownBucket) string { return item.Model }, want: []string{"large", "small"},
		},
		{
			name: "project normalized path", groupBy: "project",
			items: []model.UsageBreakdownBucket{{ProjectPath: "/workspace/b", TotalTokens: 1}, {ProjectPath: "/workspace/a", TotalTokens: 1}},
			key:   func(item model.UsageBreakdownBucket) string { return item.ProjectPath }, want: []string{"/workspace/a", "/workspace/b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortUsageBreakdownBuckets(tt.items, tt.groupBy)
			for index, want := range tt.want {
				if got := tt.key(tt.items[index]); got != want {
					t.Fatalf("item %d = %q, want %q", index, got, want)
				}
			}
		})
	}
}
