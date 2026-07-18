package pricing

import "time"

type seedSources struct {
	verified          time.Time
	verifiedLatest    time.Time
	verifiedCurrent   time.Time
	openaiPricing     string
	openaiGPT56Sol    string
	openaiGPT5        string
	openaiGPT5Mini    string
	openaiGPT5Nano    string
	anthropicPricing  string
	googlePricing     string
	deepseekPricing   string
	deepseekV4Pricing string
	zaiPricing        string
	kimiPricing       string
	mistralPricing    string
	xaiPricing        string
	coherePricing     string
	qwenPricing       string
	tencentHy3Pricing string
}

func seedRates() []Rate {
	sources := newSeedSources()
	var rates []Rate
	for _, group := range [][]Rate{
		openAISeedRates(sources),
		anthropicSeedRates(sources),
		googleSeedRates(sources),
		deepSeekSeedRates(sources),
		singleProviderSeedRates(sources),
		mistralSeedRates(sources),
		xAISeedRates(sources),
		cohereSeedRates(sources),
		qwenSeedRates(sources),
	} {
		rates = append(rates, group...)
	}
	return rates
}

func newSeedSources() seedSources {
	verified := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	verifiedLatest := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	verifiedCurrent := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	return seedSources{
		verified:          verified,
		verifiedLatest:    verifiedLatest,
		verifiedCurrent:   verifiedCurrent,
		openaiPricing:     "OpenAI API pricing, https://developers.openai.com/api/docs/pricing, verified 2026-06-27",
		openaiGPT56Sol:    "OpenAI GPT-5.6 Sol model page, https://developers.openai.com/api/docs/models/gpt-5.6-sol, verified 2026-07-17",
		openaiGPT5:        "OpenAI GPT-5 model page, https://developers.openai.com/api/docs/models/gpt-5, verified 2026-06-27",
		openaiGPT5Mini:    "OpenAI GPT-5 mini model page, https://developers.openai.com/api/docs/models/gpt-5-mini, verified 2026-06-27",
		openaiGPT5Nano:    "OpenAI GPT-5 nano model page, https://developers.openai.com/api/docs/models/gpt-5-nano, verified 2026-06-27",
		anthropicPricing:  "Anthropic Claude pricing, https://platform.claude.com/docs/en/about-claude/pricing, cache hit rate used, verified 2026-06-27",
		googlePricing:     "Google Gemini Developer API pricing, https://ai.google.dev/gemini-api/docs/pricing, standard text/image/video tier, verified 2026-06-29",
		deepseekPricing:   "DeepSeek API pricing details USD, https://api-docs.deepseek.com/quick_start/pricing-details-usd, verified 2026-06-27",
		deepseekV4Pricing: "DeepSeek API pricing, https://api-docs.deepseek.com/quick_start/pricing, verified 2026-06-27",
		zaiPricing:        "Z.AI pricing, https://docs.z.ai/guides/overview/pricing, verified 2026-06-27",
		kimiPricing:       "Kimi API Platform pricing, https://platform.kimi.ai/docs/pricing/chat-k26, verified 2026-06-27",
		mistralPricing:    "Mistral API pricing, https://mistral.ai/pricing/; cached tokens billed at 10% input per https://docs.mistral.ai/api/endpoint/chat, verified 2026-06-27",
		xaiPricing:        "xAI pricing, https://docs.x.ai/developers/pricing, verified 2026-06-27",
		coherePricing:     "Cohere pricing, https://cohere.com/pricing, no cached discount listed, verified 2026-06-27",
		qwenPricing:       "Alibaba Cloud Model Studio pricing, https://www.alibabacloud.com/help/en/model-studio/model-pricing, regional/tiered standard rates, verified 2026-06-27",
		tencentHy3Pricing: "Tencent Hy3 preview TokenHub pricing, https://www.tencent.com/en-us/articles/2202320.html, starting USD rates, verified 2026-06-29",
	}
}

func openAISeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "gpt-5.6-sol", NormalizedModel: "gpt-5.6-sol", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: s.openaiGPT56Sol, EffectiveFrom: s.verifiedCurrent},
		{Model: "gpt-5.6-terra", NormalizedModel: "gpt-5.6-terra", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.6-luna", NormalizedModel: "gpt-5.6-luna", InputPer1M: 1.00, CachedInputPer1M: 0.10, OutputPer1M: 6.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.5", NormalizedModel: "gpt-5.5", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.5-short-context", NormalizedModel: "gpt-5.5-short-context", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.5-long-context", NormalizedModel: "gpt-5.5-long-context", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 45.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.4", NormalizedModel: "gpt-5.4", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.4-short-context", NormalizedModel: "gpt-5.4-short-context", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.4-long-context", NormalizedModel: "gpt-5.4-long-context", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 22.50, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.4-mini", NormalizedModel: "gpt-5.4-mini", InputPer1M: 0.75, CachedInputPer1M: 0.075, OutputPer1M: 4.50, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.4-nano", NormalizedModel: "gpt-5.4-nano", InputPer1M: 0.20, CachedInputPer1M: 0.02, OutputPer1M: 1.25, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.3-codex", NormalizedModel: "gpt-5.3-codex", InputPer1M: 1.75, CachedInputPer1M: 0.175, OutputPer1M: 14.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5.2-codex", NormalizedModel: "gpt-5.2-codex", InputPer1M: 1.75, CachedInputPer1M: 0.175, OutputPer1M: 14.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "chat-latest", NormalizedModel: "chat-latest", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-5", NormalizedModel: "gpt-5", InputPer1M: 1.25, CachedInputPer1M: 0.125, OutputPer1M: 10.00, Source: s.openaiGPT5, EffectiveFrom: s.verified},
		{Model: "gpt-5-mini", NormalizedModel: "gpt-5-mini", InputPer1M: 0.25, CachedInputPer1M: 0.025, OutputPer1M: 2.00, Source: s.openaiGPT5Mini, EffectiveFrom: s.verified},
		{Model: "gpt-5-nano", NormalizedModel: "gpt-5-nano", InputPer1M: 0.05, CachedInputPer1M: 0.005, OutputPer1M: 0.40, Source: s.openaiGPT5Nano, EffectiveFrom: s.verified},
		{Model: "gpt-4.1", NormalizedModel: "gpt-4.1", InputPer1M: 2.00, CachedInputPer1M: 0.50, OutputPer1M: 8.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-4.1-mini", NormalizedModel: "gpt-4.1-mini", InputPer1M: 0.40, CachedInputPer1M: 0.10, OutputPer1M: 1.60, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-4.1-nano", NormalizedModel: "gpt-4.1-nano", InputPer1M: 0.10, CachedInputPer1M: 0.025, OutputPer1M: 0.40, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-4o", NormalizedModel: "gpt-4o", InputPer1M: 2.50, CachedInputPer1M: 1.25, OutputPer1M: 10.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "gpt-4o-mini", NormalizedModel: "gpt-4o-mini", InputPer1M: 0.15, CachedInputPer1M: 0.075, OutputPer1M: 0.60, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "o3", NormalizedModel: "o3", InputPer1M: 2.00, CachedInputPer1M: 0.50, OutputPer1M: 8.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "o3-pro", NormalizedModel: "o3-pro", InputPer1M: 20.00, CachedInputPer1M: 20.00, OutputPer1M: 80.00, Source: s.openaiPricing, EffectiveFrom: s.verified},
		{Model: "o4-mini", NormalizedModel: "o4-mini", InputPer1M: 1.10, CachedInputPer1M: 0.275, OutputPer1M: 4.40, Source: s.openaiPricing, EffectiveFrom: s.verified},
	}
}

func anthropicSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "claude-fable-5", NormalizedModel: "claude-fable-5", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 50.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-mythos-5", NormalizedModel: "claude-mythos-5", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 50.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4.8", NormalizedModel: "claude-opus-4.8", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4.7", NormalizedModel: "claude-opus-4.7", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4.6", NormalizedModel: "claude-opus-4.6", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4.5", NormalizedModel: "claude-opus-4.5", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4.1", NormalizedModel: "claude-opus-4.1", InputPer1M: 15.00, CachedInputPer1M: 1.50, OutputPer1M: 75.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-opus-4", NormalizedModel: "claude-opus-4", InputPer1M: 15.00, CachedInputPer1M: 1.50, OutputPer1M: 75.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-sonnet-5", NormalizedModel: "claude-sonnet-5", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 10.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-sonnet-4.6", NormalizedModel: "claude-sonnet-4.6", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-sonnet-4.6-1m", NormalizedModel: "claude-sonnet-4.6-1m", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-sonnet-4.5", NormalizedModel: "claude-sonnet-4.5", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-sonnet-4", NormalizedModel: "claude-sonnet-4", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-haiku-4.5", NormalizedModel: "claude-haiku-4.5", InputPer1M: 1.00, CachedInputPer1M: 0.10, OutputPer1M: 5.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
		{Model: "claude-haiku-3.5", NormalizedModel: "claude-haiku-3.5", InputPer1M: 0.80, CachedInputPer1M: 0.08, OutputPer1M: 4.00, Source: s.anthropicPricing, EffectiveFrom: s.verified},
	}
}

func googleSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "gemini-3.1-pro", NormalizedModel: "gemini-3.1-pro", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 12.00, Source: s.googlePricing, EffectiveFrom: s.verifiedLatest},
		{Model: "gemini-3.1-pro-preview", NormalizedModel: "gemini-3.1-pro-preview", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 12.00, Source: s.googlePricing, EffectiveFrom: s.verifiedLatest},
		{Model: "gemini-3.1-pro-long-context", NormalizedModel: "gemini-3.1-pro-long-context", InputPer1M: 4.00, CachedInputPer1M: 0.40, OutputPer1M: 18.00, Source: s.googlePricing, EffectiveFrom: s.verifiedLatest},
		{Model: "gemini-3.1-pro-preview-long-context", NormalizedModel: "gemini-3.1-pro-preview-long-context", InputPer1M: 4.00, CachedInputPer1M: 0.40, OutputPer1M: 18.00, Source: s.googlePricing, EffectiveFrom: s.verifiedLatest},
		{Model: "gemini-2.5-pro", NormalizedModel: "gemini-2.5-pro", InputPer1M: 1.25, CachedInputPer1M: 0.125, OutputPer1M: 10.00, Source: s.googlePricing, EffectiveFrom: s.verified},
		{Model: "gemini-2.5-pro-long-context", NormalizedModel: "gemini-2.5-pro-long-context", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: s.googlePricing, EffectiveFrom: s.verified},
		{Model: "gemini-2.5-flash", NormalizedModel: "gemini-2.5-flash", InputPer1M: 0.30, CachedInputPer1M: 0.03, OutputPer1M: 2.50, Source: s.googlePricing, EffectiveFrom: s.verified},
		{Model: "gemini-2.5-flash-lite", NormalizedModel: "gemini-2.5-flash-lite", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.40, Source: s.googlePricing, EffectiveFrom: s.verified},
		{Model: "gemini-2.0-flash", NormalizedModel: "gemini-2.0-flash", InputPer1M: 0.10, CachedInputPer1M: 0.025, OutputPer1M: 0.40, Source: s.googlePricing, EffectiveFrom: s.verified},
	}
}

func deepSeekSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "deepseek-chat", NormalizedModel: "deepseek-chat", InputPer1M: 0.27, CachedInputPer1M: 0.07, OutputPer1M: 1.10, Source: s.deepseekPricing, EffectiveFrom: s.verified},
		{Model: "deepseek-reasoner", NormalizedModel: "deepseek-reasoner", InputPer1M: 0.55, CachedInputPer1M: 0.14, OutputPer1M: 2.19, Source: s.deepseekPricing, EffectiveFrom: s.verified},
		{Model: "deepseek-v4-flash", NormalizedModel: "deepseek-v4-flash", InputPer1M: 0.14, CachedInputPer1M: 0.0028, OutputPer1M: 0.28, Source: s.deepseekV4Pricing, EffectiveFrom: s.verified},
		{Model: "deepseek-v4-pro", NormalizedModel: "deepseek-v4-pro", InputPer1M: 0.435, CachedInputPer1M: 0.003625, OutputPer1M: 0.87, Source: s.deepseekV4Pricing, EffectiveFrom: s.verified},
	}
}

func singleProviderSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "hy3-preview", NormalizedModel: "hy3-preview", InputPer1M: 0.18, CachedInputPer1M: 0.06, OutputPer1M: 0.59, Source: s.tencentHy3Pricing, EffectiveFrom: s.verifiedLatest},
		{Model: "glm-5.2", NormalizedModel: "glm-5.2", InputPer1M: 1.40, CachedInputPer1M: 0.26, OutputPer1M: 4.40, Source: s.zaiPricing, EffectiveFrom: s.verified},
		{Model: "kimi-k2.6", NormalizedModel: "kimi-k2.6", InputPer1M: 0.95, CachedInputPer1M: 0.16, OutputPer1M: 4.00, Source: s.kimiPricing, EffectiveFrom: s.verified},
	}
}

func mistralSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "mistral-medium-latest", NormalizedModel: "mistral-medium-latest", InputPer1M: 1.50, CachedInputPer1M: 0.15, OutputPer1M: 7.50, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "mistral-small-latest", NormalizedModel: "mistral-small-latest", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.60, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "mistral-large-latest", NormalizedModel: "mistral-large-latest", InputPer1M: 0.50, CachedInputPer1M: 0.05, OutputPer1M: 1.50, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "devstral-medium-latest", NormalizedModel: "devstral-medium-latest", InputPer1M: 0.40, CachedInputPer1M: 0.04, OutputPer1M: 2.00, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "devstral-small-latest", NormalizedModel: "devstral-small-latest", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.30, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "codestral-latest", NormalizedModel: "codestral-latest", InputPer1M: 0.30, CachedInputPer1M: 0.03, OutputPer1M: 0.90, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "magistral-medium-latest", NormalizedModel: "magistral-medium-latest", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 5.00, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "magistral-small-latest", NormalizedModel: "magistral-small-latest", InputPer1M: 0.50, CachedInputPer1M: 0.05, OutputPer1M: 1.50, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "ministral-3b-latest", NormalizedModel: "ministral-3b-latest", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.10, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "ministral-8b-latest", NormalizedModel: "ministral-8b-latest", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.15, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "ministral-14b-latest", NormalizedModel: "ministral-14b-latest", InputPer1M: 0.20, CachedInputPer1M: 0.02, OutputPer1M: 0.20, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "open-mistral-nemo", NormalizedModel: "open-mistral-nemo", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.15, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "open-mixtral-8x7b", NormalizedModel: "open-mixtral-8x7b", InputPer1M: 0.70, CachedInputPer1M: 0.07, OutputPer1M: 0.70, Source: s.mistralPricing, EffectiveFrom: s.verified},
		{Model: "open-mixtral-8x22b", NormalizedModel: "open-mixtral-8x22b", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 6.00, Source: s.mistralPricing, EffectiveFrom: s.verified},
	}
}

func xAISeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "grok-4.3", NormalizedModel: "grok-4.3", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: s.xaiPricing, EffectiveFrom: s.verified},
		{Model: "grok-4.20-multi-agent-0309", NormalizedModel: "grok-4.20-multi-agent-0309", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: s.xaiPricing, EffectiveFrom: s.verified},
		{Model: "grok-4.20-0309-reasoning", NormalizedModel: "grok-4.20-0309-reasoning", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: s.xaiPricing, EffectiveFrom: s.verified},
		{Model: "grok-4.20-0309-non-reasoning", NormalizedModel: "grok-4.20-0309-non-reasoning", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: s.xaiPricing, EffectiveFrom: s.verified},
		{Model: "grok-build-0.1", NormalizedModel: "grok-build-0.1", InputPer1M: 1.00, CachedInputPer1M: 0.20, OutputPer1M: 2.00, Source: s.xaiPricing, EffectiveFrom: s.verified},
	}
}

func cohereSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "command", NormalizedModel: "command", InputPer1M: 1.00, CachedInputPer1M: 1.00, OutputPer1M: 2.00, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "command-light", NormalizedModel: "command-light", InputPer1M: 0.30, CachedInputPer1M: 0.30, OutputPer1M: 0.60, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "command-r", NormalizedModel: "command-r", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "command-r-plus", NormalizedModel: "command-r-plus", InputPer1M: 2.50, CachedInputPer1M: 2.50, OutputPer1M: 10.00, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "command-r-plus-04-2024", NormalizedModel: "command-r-plus-04-2024", InputPer1M: 3.00, CachedInputPer1M: 3.00, OutputPer1M: 15.00, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "command-r-plus-08-2024", NormalizedModel: "command-r-plus-08-2024", InputPer1M: 2.50, CachedInputPer1M: 2.50, OutputPer1M: 10.00, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "aya-expanse-8b", NormalizedModel: "aya-expanse-8b", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: s.coherePricing, EffectiveFrom: s.verified},
		{Model: "aya-expanse-32b", NormalizedModel: "aya-expanse-32b", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: s.coherePricing, EffectiveFrom: s.verified},
	}
}

func qwenSeedRates(s seedSources) []Rate {
	return []Rate{
		{Model: "qwen3-max", NormalizedModel: "qwen3-max", InputPer1M: 1.65, CachedInputPer1M: 1.65, OutputPer1M: 4.951, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen-max", NormalizedModel: "qwen-max", InputPer1M: 1.60, CachedInputPer1M: 1.60, OutputPer1M: 6.40, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen3.6-plus", NormalizedModel: "qwen3.6-plus", InputPer1M: 0.276, CachedInputPer1M: 0.276, OutputPer1M: 1.651, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen-plus", NormalizedModel: "qwen-plus", InputPer1M: 0.40, CachedInputPer1M: 0.40, OutputPer1M: 4.00, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen-turbo", NormalizedModel: "qwen-turbo", InputPer1M: 0.05, CachedInputPer1M: 0.05, OutputPer1M: 0.50, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen3-coder-plus", NormalizedModel: "qwen3-coder-plus", InputPer1M: 0.574, CachedInputPer1M: 0.574, OutputPer1M: 2.294, Source: s.qwenPricing, EffectiveFrom: s.verified},
		{Model: "qwen3-coder-flash", NormalizedModel: "qwen3-coder-flash", InputPer1M: 0.144, CachedInputPer1M: 0.144, OutputPer1M: 0.574, Source: s.qwenPricing, EffectiveFrom: s.verified},
	}
}
