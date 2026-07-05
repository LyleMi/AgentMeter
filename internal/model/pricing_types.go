package model

import "time"

type PricingModel struct {
	ID               int64     `json:"id"`
	Model            string    `json:"model"`
	NormalizedModel  string    `json:"normalizedModel"`
	InputPer1M       float64   `json:"inputPer1m"`
	CachedInputPer1M float64   `json:"cachedInputPer1m"`
	OutputPer1M      float64   `json:"outputPer1m"`
	Source           string    `json:"source"`
	EffectiveFrom    time.Time `json:"effectiveFrom"`
	IsCustom         bool      `json:"isCustom"`
}

type PricingModelInput struct {
	Model            string  `json:"model"`
	InputPer1M       float64 `json:"inputPer1m"`
	CachedInputPer1M float64 `json:"cachedInputPer1m"`
	OutputPer1M      float64 `json:"outputPer1m"`
	Source           string  `json:"source,omitempty"`
}
