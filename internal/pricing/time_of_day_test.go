package pricing

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestDeepSeekPeakBoundaries(t *testing.T) {
	tests := []struct {
		at  string
		off bool
	}{
		{"2026-09-21T08:59:59+08:00", true},
		{"2026-09-21T09:00:00+08:00", false},
		{"2026-09-21T11:59:59+08:00", false},
		{"2026-09-21T12:00:00+08:00", true},
		{"2026-09-21T13:59:59+08:00", true},
		{"2026-09-21T14:00:00+08:00", false},
		{"2026-09-21T17:59:59+08:00", false},
		{"2026-09-21T18:00:00+08:00", true},
		{"2026-09-21T01:00:00Z", false},
		{"2026-09-20T21:00:00-04:00", false},
		{"2026-09-19T09:00:00+08:00", true},
		{"2026-09-20T09:00:00+08:00", true}, // Sunday make-up workday is still off-peak.
		{"2026-09-25T09:00:00+08:00", true}, // Mid-Autumn holiday on Friday.
		{"2026-10-01T14:00:00+08:00", true},
		{"2026-10-08T14:00:00+08:00", false},
		{"2027-10-01T09:00:00+08:00", false}, // Holiday calendar not yet verified.
	}
	for _, tt := range tests {
		at, err := time.Parse(time.RFC3339, tt.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := deepSeekOffPeak(at); got != tt.off {
			t.Errorf("%s off-peak = %v, want %v", tt.at, got, tt.off)
		}
	}
	if deepSeekOffPeak(time.Time{}) {
		t.Fatal("missing time must not receive a discount")
	}
}

func TestTimedPricingRespectsCustomAndExplicitRates(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()
	at, _ := time.Parse(time.RFC3339, "2026-09-21T04:00:00Z")
	usage := model.Usage{Model: "deepseek/deepseek-v4-pro-0813", PricingTime: at, InputTokens: 1_000_000, CachedInputTokens: 200_000, OutputTokens: 1_000_000}
	assertCost := func(want float64) {
		t.Helper()
		calc, err := LoadCalculator(context.Background(), conn)
		if err != nil {
			t.Fatal(err)
		}
		direct, missing := Compute(conn, usage)
		cached, missingCached := calc.Compute(usage)
		if missing || missingCached || direct == nil || cached == nil || math.Abs(*direct-want) > 1e-9 || math.Abs(*cached-want) > 1e-9 {
			t.Fatalf("cost=%v cached=%v want=%v", direct, cached, want)
		}
	}
	assertCost(.8*.66 + .2*.022 + 1.98)
	usage.Model = "deepseek-v4-pro-off-peak"
	assertCost(.8*.66 + .2*.022 + 1.98) // Must not halve explicit off-peak a second time.
	usage.Model = "deepseek-v4-pro"
	_, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{Model: usage.Model, InputPer1M: 10, CachedInputPer1M: 2, OutputPer1M: 20})
	if err != nil {
		t.Fatal(err)
	}
	assertCost(28.4)
}

func TestCallTimeUsesStartAcrossBoundary(t *testing.T) {
	start, _ := time.Parse(time.RFC3339, "2026-09-21T03:59:59Z")
	end := start.Add(2 * time.Minute)
	if deepSeekOffPeak(CallTime(start, end)) {
		t.Fatal("request starting at peak must retain peak rate")
	}
	if !deepSeekOffPeak(CallTime(time.Time{}, end)) {
		t.Fatal("missing start must use completion time")
	}
}
