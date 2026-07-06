package model

import (
	"math"
	"testing"
)

func TestRiskThresholdScore(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		warning  float64
		critical float64
		want     float64
	}{
		{name: "below warning", value: 10, warning: 20, critical: 40, want: 0},
		{name: "between thresholds", value: 30, warning: 20, critical: 40, want: 0.5},
		{name: "at critical", value: 40, warning: 20, critical: 40, want: 1},
		{name: "invalid thresholds", value: 30, warning: 40, critical: 20, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RiskThresholdScore(tt.value, tt.warning, tt.critical)
			if got != tt.want {
				t.Fatalf("RiskThresholdScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInverseRiskThresholdScore(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		warning  float64
		critical float64
		want     float64
	}{
		{name: "above warning", value: 50, warning: 40, critical: 10, want: 0},
		{name: "between thresholds", value: 25, warning: 40, critical: 10, want: 0.5},
		{name: "at critical", value: 10, warning: 40, critical: 10, want: 1},
		{name: "invalid thresholds", value: 25, warning: 10, critical: 40, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InverseRiskThresholdScore(tt.value, tt.warning, tt.critical)
			if got != tt.want {
				t.Fatalf("InverseRiskThresholdScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRiskRangeScore(t *testing.T) {
	if got := RiskRangeScore(7, 5, 10); got != 0.2 {
		t.Fatalf("RiskRangeScore() = %v, want 0.2", got)
	}
	if got := RiskRangeScore(20, 5, 10); got != 1 {
		t.Fatalf("RiskRangeScore() = %v, want 1", got)
	}
	if got := RiskRangeScore(7, 5, 0); got != 0 {
		t.Fatalf("RiskRangeScore() with zero span = %v, want 0", got)
	}
}

func TestClampRiskScore(t *testing.T) {
	for _, value := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := ClampRiskScore(value); got != 0 {
			t.Fatalf("ClampRiskScore(%v) = %v, want 0", value, got)
		}
	}
	if got := ClampRiskScore(2); got != 1 {
		t.Fatalf("ClampRiskScore(2) = %v, want 1", got)
	}
}
