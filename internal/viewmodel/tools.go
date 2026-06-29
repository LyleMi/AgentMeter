package viewmodel

import (
	"math"
	"sort"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type ToolSummary struct {
	TotalCalls              int     `json:"totalCalls"`
	TotalCallsLabel         string  `json:"totalCallsLabel"`
	ToolsUsed               int     `json:"toolsUsed"`
	ToolsUsedLabel          string  `json:"toolsUsedLabel"`
	FailedPendingCalls      int     `json:"failedPendingCalls"`
	FailedPendingCallsLabel string  `json:"failedPendingCallsLabel"`
	TotalDurationMS         int64   `json:"totalDurationMs"`
	AverageDurationMS       float64 `json:"averageDurationMs"`
	AverageDurationLabel    string  `json:"averageDurationLabel"`
	DurationSignal          Signal  `json:"durationSignal"`
}

func SummarizeTools(stats []model.ToolStat) ToolSummary {
	var summary ToolSummary
	summary.ToolsUsed = len(stats)
	for _, item := range stats {
		summary.TotalCalls += item.Calls
		summary.FailedPendingCalls += item.FailedCalls
		summary.TotalDurationMS += item.TotalDurationMS
	}
	if summary.TotalCalls > 0 {
		summary.AverageDurationMS = float64(summary.TotalDurationMS) / float64(summary.TotalCalls)
	}
	summary.TotalCallsLabel = FormatNumber(int64(summary.TotalCalls))
	summary.ToolsUsedLabel = FormatNumber(int64(summary.ToolsUsed))
	summary.FailedPendingCallsLabel = FormatNumber(int64(summary.FailedPendingCalls))
	summary.AverageDurationLabel = FormatDuration(summary.AverageDurationMS)
	summary.DurationSignal = ToolDurationSignal(summary.AverageDurationMS, summary.TotalCalls)
	return summary
}

func ToolSuccessRate(stat model.ToolStat) int {
	if stat.Calls <= 0 {
		return 0
	}
	return int(math.Round((float64(stat.SuccessCalls) / float64(stat.Calls)) * 100))
}

func ToolSuccessStatus(stat model.ToolStat) Signal {
	rate := ToolSuccessRate(stat)
	if stat.Calls <= 0 {
		return Signal{Tone: ToneDefault, Label: "No calls"}
	}
	label := FormatNumber(int64(rate)) + "% ok"
	if rate >= 99 {
		return Signal{Tone: ToneSuccess, Label: label}
	}
	if rate >= 90 {
		return Signal{Tone: ToneWarning, Label: label}
	}
	return Signal{Tone: ToneError, Label: label}
}

func ToolFailureStatus(stat model.ToolStat) Signal {
	if stat.FailedCalls <= 0 {
		return Signal{Tone: ToneSuccess, Label: "Clear"}
	}
	rate := int(math.Round((float64(stat.FailedCalls) / math.Max(float64(stat.Calls), 1)) * 100))
	tone := ToneWarning
	if rate >= 10 {
		tone = ToneError
	}
	return Signal{Tone: tone, Label: FormatNumber(int64(stat.FailedCalls)) + " affected"}
}

func ToolDurationSignal(averageDurationMS float64, calls int) Signal {
	if calls <= 0 {
		return Signal{Tone: ToneDefault, Label: "No calls"}
	}
	if averageDurationMS >= 60000 {
		return Signal{Tone: ToneWarning, Label: "Long average"}
	}
	if averageDurationMS >= 10000 {
		return Signal{Tone: ToneProcessing, Label: "Moderate average"}
	}
	return Signal{Tone: ToneSuccess, Label: "Fast average"}
}

func TopToolsByCalls(stats []model.ToolStat, limit int) []model.ToolStat {
	if limit <= 0 {
		return nil
	}
	ranked := append([]model.ToolStat(nil), stats...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Calls == ranked[j].Calls {
			return ranked[i].ToolName < ranked[j].ToolName
		}
		return ranked[i].Calls > ranked[j].Calls
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}
