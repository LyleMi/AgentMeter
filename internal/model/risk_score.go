package model

import "math"

// RiskThresholdScore scores values where higher is worse.
func RiskThresholdScore(value, warning, critical float64) float64 {
	if value <= warning || warning >= critical {
		return 0
	}
	if value >= critical {
		return 1
	}
	return ClampRiskScore((value - warning) / (critical - warning))
}

// InverseRiskThresholdScore scores values where lower is worse.
func InverseRiskThresholdScore(value, warning, critical float64) float64 {
	if value <= 0 || warning <= critical {
		return 0
	}
	if value >= warning {
		return 0
	}
	if value <= critical {
		return 1
	}
	return ClampRiskScore((warning - value) / (warning - critical))
}

// RiskRangeScore scores values above start across a fixed span.
func RiskRangeScore(value, start, span float64) float64 {
	if value <= start || span <= 0 {
		return 0
	}
	return ClampRiskScore((value - start) / span)
}

// ClampRiskScore keeps risk contributions in the normalized 0..1 range.
func ClampRiskScore(value float64) float64 {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
