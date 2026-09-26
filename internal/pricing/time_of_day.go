package pricing

import (
	"context"
	"database/sql"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

var beijing = time.FixedZone("Asia/Shanghai", 8*60*60)

// Published Chinese holiday periods (including adjacent days off), 2026:
// https://www.gov.cn/gongbao/2025/issue_12406/202511/content_7048922.html
// Weekend make-up workdays remain off-peak under DeepSeek's Monday–Friday rule.
var holidays2026 = [][2]string{
	{"2026-01-01", "2026-01-03"}, {"2026-02-15", "2026-02-23"},
	{"2026-04-04", "2026-04-06"}, {"2026-05-01", "2026-05-05"},
	{"2026-06-19", "2026-06-21"}, {"2026-09-25", "2026-09-27"},
	{"2026-10-01", "2026-10-07"},
}

func deepSeekOffPeak(at time.Time) bool {
	if at.IsZero() {
		return false
	}
	local := at.In(beijing)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return true
	}
	hour := local.Hour()
	if !(hour >= 9 && hour < 12 || hour >= 14 && hour < 18) {
		return true
	}
	// Unknown holiday years conservatively retain peak pricing during weekday
	// peak windows. Night/weekend discounts do not depend on the calendar table.
	day := local.Format("2006-01-02")
	for _, period := range holidays2026 {
		if day >= period[0] && day <= period[1] {
			return true
		}
	}
	return false
}

func usesTimeOfDay(rate Rate) bool {
	if rate.IsCustom {
		return false
	}
	switch rate.NormalizedModel {
	case "deepseek-flash", "deepseek-v4-flash", "deepseek-v4.1-flash", "deepseek-v4-pro":
		return true
	}
	return false
}

func rateAt(rate Rate, at time.Time) Rate {
	// Explicit off-peak aliases and user overrides are already final rates.
	if usesTimeOfDay(rate) && deepSeekOffPeak(at) {
		rate.InputPer1M /= 2
		rate.CachedInputPer1M /= 2
		rate.OutputPer1M /= 2
	}
	return rate
}

// CallTime uses request start as the local proxy for provider receipt time.
// Completion time is the fallback when the log omits the start; never split
// one request's tokens by elapsed time across a tariff boundary.
func CallTime(start, end time.Time) time.Time {
	if !start.IsZero() {
		return start
	}
	return end
}

// LoadSessionCalculator supplements the registry with per-call accounting for
// sessions containing DeepSeek. Other sessions retain aggregate calculation.
// Keeping all calls in a matching session also handles model switches.
func LoadSessionCalculator(ctx context.Context, conn *sql.DB) (Calculator, error) {
	calc, err := LoadCalculator(ctx, conn)
	if err != nil {
		return calc, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT session_id, model, started_at, ended_at,
 input_tokens, cached_input_tokens, output_tokens, reasoning_output_tokens, total_tokens
 FROM model_calls WHERE session_id IN
 (SELECT session_id FROM model_calls WHERE lower(model) LIKE '%deepseek%') ORDER BY session_id, id`)
	if err != nil {
		return calc, err
	}
	defer rows.Close()
	calc.sessionCalls = map[int64][]model.Usage{}
	for rows.Next() {
		var id int64
		var usage model.Usage
		var start, end string
		if err := rows.Scan(&id, &usage.Model, &start, &end, &usage.InputTokens, &usage.CachedInputTokens, &usage.OutputTokens, &usage.ReasoningOutputTokens, &usage.TotalTokens); err != nil {
			return calc, err
		}
		started, _ := time.Parse(time.RFC3339Nano, start)
		ended, _ := time.Parse(time.RFC3339Nano, end)
		usage.PricingTime = CallTime(started, ended)
		calc.sessionCalls[id] = append(calc.sessionCalls[id], usage)
	}
	return calc, rows.Err()
}

func (c Calculator) coveredSessionCalls(usage model.Usage) ([]model.Usage, bool) {
	calls := c.sessionCalls[usage.PricingSessionID]
	if len(calls) == 0 {
		return nil, false
	}
	var input, cached, output, reasoning, total int64
	for _, call := range calls {
		input += call.InputTokens
		cached += call.CachedInputTokens
		output += call.OutputTokens
		reasoning += call.ReasoningOutputTokens
		total += call.TotalTokens
	}
	// Partial/duplicated call histories must not silently under/overcount the
	// authoritative session counters. Fall back to session-time estimation.
	return calls, input == usage.InputTokens && cached == usage.CachedInputTokens && output == usage.OutputTokens && reasoning == usage.ReasoningOutputTokens && total == usage.TotalTokens
}
