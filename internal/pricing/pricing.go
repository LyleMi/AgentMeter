package pricing

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type Rate struct {
	Model            string
	NormalizedModel  string
	InputPer1M       float64
	CachedInputPer1M float64
	OutputPer1M      float64
	Source           string
	EffectiveFrom    time.Time
	IsCustom         bool
}

type Calculator struct {
	rates map[string]Rate
}

var ErrInvalidRate = errors.New("invalid pricing model")

func Seed(ctx context.Context, conn *sql.DB) error {
	// These are API list-price estimates used only when a local Codex model name
	// can be matched. Codex subscription usage may not map one-to-one to API cost.
	verified := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	verifiedLatest := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	openaiPricing := "OpenAI API pricing, https://developers.openai.com/api/docs/pricing, verified 2026-06-27"
	openaiGPT5 := "OpenAI GPT-5 model page, https://developers.openai.com/api/docs/models/gpt-5, verified 2026-06-27"
	openaiGPT5Mini := "OpenAI GPT-5 mini model page, https://developers.openai.com/api/docs/models/gpt-5-mini, verified 2026-06-27"
	openaiGPT5Nano := "OpenAI GPT-5 nano model page, https://developers.openai.com/api/docs/models/gpt-5-nano, verified 2026-06-27"
	anthropicPricing := "Anthropic Claude pricing, https://platform.claude.com/docs/en/about-claude/pricing, cache hit rate used, verified 2026-06-27"
	googlePricing := "Google Gemini Developer API pricing, https://ai.google.dev/gemini-api/docs/pricing, standard text/image/video tier, verified 2026-06-29"
	deepseekPricing := "DeepSeek API pricing details USD, https://api-docs.deepseek.com/quick_start/pricing-details-usd, verified 2026-06-27"
	deepseekV4Pricing := "DeepSeek API pricing, https://api-docs.deepseek.com/quick_start/pricing, verified 2026-06-27"
	zaiPricing := "Z.AI pricing, https://docs.z.ai/guides/overview/pricing, verified 2026-06-27"
	kimiPricing := "Kimi API Platform pricing, https://platform.kimi.ai/docs/pricing/chat-k26, verified 2026-06-27"
	mistralPricing := "Mistral API pricing, https://mistral.ai/pricing/; cached tokens billed at 10% input per https://docs.mistral.ai/api/endpoint/chat, verified 2026-06-27"
	xaiPricing := "xAI pricing, https://docs.x.ai/developers/pricing, verified 2026-06-27"
	coherePricing := "Cohere pricing, https://cohere.com/pricing, no cached discount listed, verified 2026-06-27"
	qwenPricing := "Alibaba Cloud Model Studio pricing, https://www.alibabacloud.com/help/en/model-studio/model-pricing, regional/tiered standard rates, verified 2026-06-27"
	tencentHy3Pricing := "Tencent Hy3 preview TokenHub pricing, https://www.tencent.com/en-us/articles/2202320.html, starting USD rates, verified 2026-06-29"
	rates := []Rate{
		// OpenAI. Long-context rows are explicit opt-in aliases because the
		// session JSONL model name does not expose prompt length by itself.
		{Model: "gpt-5.5", NormalizedModel: "gpt-5.5", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.5-short-context", NormalizedModel: "gpt-5.5-short-context", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.5-long-context", NormalizedModel: "gpt-5.5-long-context", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 45.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.4", NormalizedModel: "gpt-5.4", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.4-short-context", NormalizedModel: "gpt-5.4-short-context", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.4-long-context", NormalizedModel: "gpt-5.4-long-context", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 22.50, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.4-mini", NormalizedModel: "gpt-5.4-mini", InputPer1M: 0.75, CachedInputPer1M: 0.075, OutputPer1M: 4.50, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.4-nano", NormalizedModel: "gpt-5.4-nano", InputPer1M: 0.20, CachedInputPer1M: 0.02, OutputPer1M: 1.25, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5.3-codex", NormalizedModel: "gpt-5.3-codex", InputPer1M: 1.75, CachedInputPer1M: 0.175, OutputPer1M: 14.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "chat-latest", NormalizedModel: "chat-latest", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-5", NormalizedModel: "gpt-5", InputPer1M: 1.25, CachedInputPer1M: 0.125, OutputPer1M: 10.00, Source: openaiGPT5, EffectiveFrom: verified},
		{Model: "gpt-5-mini", NormalizedModel: "gpt-5-mini", InputPer1M: 0.25, CachedInputPer1M: 0.025, OutputPer1M: 2.00, Source: openaiGPT5Mini, EffectiveFrom: verified},
		{Model: "gpt-5-nano", NormalizedModel: "gpt-5-nano", InputPer1M: 0.05, CachedInputPer1M: 0.005, OutputPer1M: 0.40, Source: openaiGPT5Nano, EffectiveFrom: verified},
		{Model: "gpt-4.1", NormalizedModel: "gpt-4.1", InputPer1M: 2.00, CachedInputPer1M: 0.50, OutputPer1M: 8.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-4.1-mini", NormalizedModel: "gpt-4.1-mini", InputPer1M: 0.40, CachedInputPer1M: 0.10, OutputPer1M: 1.60, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-4.1-nano", NormalizedModel: "gpt-4.1-nano", InputPer1M: 0.10, CachedInputPer1M: 0.025, OutputPer1M: 0.40, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-4o", NormalizedModel: "gpt-4o", InputPer1M: 2.50, CachedInputPer1M: 1.25, OutputPer1M: 10.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "gpt-4o-mini", NormalizedModel: "gpt-4o-mini", InputPer1M: 0.15, CachedInputPer1M: 0.075, OutputPer1M: 0.60, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "o3", NormalizedModel: "o3", InputPer1M: 2.00, CachedInputPer1M: 0.50, OutputPer1M: 8.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "o3-pro", NormalizedModel: "o3-pro", InputPer1M: 20.00, CachedInputPer1M: 20.00, OutputPer1M: 80.00, Source: openaiPricing, EffectiveFrom: verified},
		{Model: "o4-mini", NormalizedModel: "o4-mini", InputPer1M: 1.10, CachedInputPer1M: 0.275, OutputPer1M: 4.40, Source: openaiPricing, EffectiveFrom: verified},

		// Anthropic Claude. Cached input uses cache-hit/read pricing; cache
		// creation write premiums are not separately represented in this schema.
		{Model: "claude-fable-5", NormalizedModel: "claude-fable-5", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 50.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-mythos-5", NormalizedModel: "claude-mythos-5", InputPer1M: 10.00, CachedInputPer1M: 1.00, OutputPer1M: 50.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4.8", NormalizedModel: "claude-opus-4.8", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4.7", NormalizedModel: "claude-opus-4.7", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4.6", NormalizedModel: "claude-opus-4.6", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4.5", NormalizedModel: "claude-opus-4.5", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 25.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4.1", NormalizedModel: "claude-opus-4.1", InputPer1M: 15.00, CachedInputPer1M: 1.50, OutputPer1M: 75.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-opus-4", NormalizedModel: "claude-opus-4", InputPer1M: 15.00, CachedInputPer1M: 1.50, OutputPer1M: 75.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-sonnet-4.6", NormalizedModel: "claude-sonnet-4.6", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-sonnet-4.6-1m", NormalizedModel: "claude-sonnet-4.6-1m", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-sonnet-4.5", NormalizedModel: "claude-sonnet-4.5", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-sonnet-4", NormalizedModel: "claude-sonnet-4", InputPer1M: 3.00, CachedInputPer1M: 0.30, OutputPer1M: 15.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-haiku-4.5", NormalizedModel: "claude-haiku-4.5", InputPer1M: 1.00, CachedInputPer1M: 0.10, OutputPer1M: 5.00, Source: anthropicPricing, EffectiveFrom: verified},
		{Model: "claude-haiku-3.5", NormalizedModel: "claude-haiku-3.5", InputPer1M: 0.80, CachedInputPer1M: 0.08, OutputPer1M: 4.00, Source: anthropicPricing, EffectiveFrom: verified},

		// Google Gemini.
		{Model: "gemini-3.1-pro", NormalizedModel: "gemini-3.1-pro", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 12.00, Source: googlePricing, EffectiveFrom: verifiedLatest},
		{Model: "gemini-3.1-pro-preview", NormalizedModel: "gemini-3.1-pro-preview", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 12.00, Source: googlePricing, EffectiveFrom: verifiedLatest},
		{Model: "gemini-3.1-pro-long-context", NormalizedModel: "gemini-3.1-pro-long-context", InputPer1M: 4.00, CachedInputPer1M: 0.40, OutputPer1M: 18.00, Source: googlePricing, EffectiveFrom: verifiedLatest},
		{Model: "gemini-3.1-pro-preview-long-context", NormalizedModel: "gemini-3.1-pro-preview-long-context", InputPer1M: 4.00, CachedInputPer1M: 0.40, OutputPer1M: 18.00, Source: googlePricing, EffectiveFrom: verifiedLatest},
		{Model: "gemini-2.5-pro", NormalizedModel: "gemini-2.5-pro", InputPer1M: 1.25, CachedInputPer1M: 0.125, OutputPer1M: 10.00, Source: googlePricing, EffectiveFrom: verified},
		{Model: "gemini-2.5-pro-long-context", NormalizedModel: "gemini-2.5-pro-long-context", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: googlePricing, EffectiveFrom: verified},
		{Model: "gemini-2.5-flash", NormalizedModel: "gemini-2.5-flash", InputPer1M: 0.30, CachedInputPer1M: 0.03, OutputPer1M: 2.50, Source: googlePricing, EffectiveFrom: verified},
		{Model: "gemini-2.5-flash-lite", NormalizedModel: "gemini-2.5-flash-lite", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.40, Source: googlePricing, EffectiveFrom: verified},
		{Model: "gemini-2.0-flash", NormalizedModel: "gemini-2.0-flash", InputPer1M: 0.10, CachedInputPer1M: 0.025, OutputPer1M: 0.40, Source: googlePricing, EffectiveFrom: verified},

		// Tencent Hunyuan / Hy.
		{Model: "hy3-preview", NormalizedModel: "hy3-preview", InputPer1M: 0.18, CachedInputPer1M: 0.06, OutputPer1M: 0.59, Source: tencentHy3Pricing, EffectiveFrom: verifiedLatest},

		// DeepSeek.
		{Model: "deepseek-chat", NormalizedModel: "deepseek-chat", InputPer1M: 0.27, CachedInputPer1M: 0.07, OutputPer1M: 1.10, Source: deepseekPricing, EffectiveFrom: verified},
		{Model: "deepseek-reasoner", NormalizedModel: "deepseek-reasoner", InputPer1M: 0.55, CachedInputPer1M: 0.14, OutputPer1M: 2.19, Source: deepseekPricing, EffectiveFrom: verified},
		{Model: "deepseek-v4-flash", NormalizedModel: "deepseek-v4-flash", InputPer1M: 0.14, CachedInputPer1M: 0.0028, OutputPer1M: 0.28, Source: deepseekV4Pricing, EffectiveFrom: verified},
		{Model: "deepseek-v4-pro", NormalizedModel: "deepseek-v4-pro", InputPer1M: 0.435, CachedInputPer1M: 0.003625, OutputPer1M: 0.87, Source: deepseekV4Pricing, EffectiveFrom: verified},

		// Z.AI / GLM.
		{Model: "glm-5.2", NormalizedModel: "glm-5.2", InputPer1M: 1.40, CachedInputPer1M: 0.26, OutputPer1M: 4.40, Source: zaiPricing, EffectiveFrom: verified},

		// Moonshot AI / Kimi.
		{Model: "kimi-k2.6", NormalizedModel: "kimi-k2.6", InputPer1M: 0.95, CachedInputPer1M: 0.16, OutputPer1M: 4.00, Source: kimiPricing, EffectiveFrom: verified},

		// Mistral.
		{Model: "mistral-medium-latest", NormalizedModel: "mistral-medium-latest", InputPer1M: 1.50, CachedInputPer1M: 0.15, OutputPer1M: 7.50, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "mistral-small-latest", NormalizedModel: "mistral-small-latest", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.60, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "mistral-large-latest", NormalizedModel: "mistral-large-latest", InputPer1M: 0.50, CachedInputPer1M: 0.05, OutputPer1M: 1.50, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "devstral-medium-latest", NormalizedModel: "devstral-medium-latest", InputPer1M: 0.40, CachedInputPer1M: 0.04, OutputPer1M: 2.00, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "devstral-small-latest", NormalizedModel: "devstral-small-latest", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.30, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "codestral-latest", NormalizedModel: "codestral-latest", InputPer1M: 0.30, CachedInputPer1M: 0.03, OutputPer1M: 0.90, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "magistral-medium-latest", NormalizedModel: "magistral-medium-latest", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 5.00, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "magistral-small-latest", NormalizedModel: "magistral-small-latest", InputPer1M: 0.50, CachedInputPer1M: 0.05, OutputPer1M: 1.50, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "ministral-3b-latest", NormalizedModel: "ministral-3b-latest", InputPer1M: 0.10, CachedInputPer1M: 0.01, OutputPer1M: 0.10, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "ministral-8b-latest", NormalizedModel: "ministral-8b-latest", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.15, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "ministral-14b-latest", NormalizedModel: "ministral-14b-latest", InputPer1M: 0.20, CachedInputPer1M: 0.02, OutputPer1M: 0.20, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "open-mistral-nemo", NormalizedModel: "open-mistral-nemo", InputPer1M: 0.15, CachedInputPer1M: 0.015, OutputPer1M: 0.15, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "open-mixtral-8x7b", NormalizedModel: "open-mixtral-8x7b", InputPer1M: 0.70, CachedInputPer1M: 0.07, OutputPer1M: 0.70, Source: mistralPricing, EffectiveFrom: verified},
		{Model: "open-mixtral-8x22b", NormalizedModel: "open-mixtral-8x22b", InputPer1M: 2.00, CachedInputPer1M: 0.20, OutputPer1M: 6.00, Source: mistralPricing, EffectiveFrom: verified},

		// xAI.
		{Model: "grok-4.3", NormalizedModel: "grok-4.3", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: xaiPricing, EffectiveFrom: verified},
		{Model: "grok-4.20-multi-agent-0309", NormalizedModel: "grok-4.20-multi-agent-0309", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: xaiPricing, EffectiveFrom: verified},
		{Model: "grok-4.20-0309-reasoning", NormalizedModel: "grok-4.20-0309-reasoning", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: xaiPricing, EffectiveFrom: verified},
		{Model: "grok-4.20-0309-non-reasoning", NormalizedModel: "grok-4.20-0309-non-reasoning", InputPer1M: 1.25, CachedInputPer1M: 0.20, OutputPer1M: 2.50, Source: xaiPricing, EffectiveFrom: verified},
		{Model: "grok-build-0.1", NormalizedModel: "grok-build-0.1", InputPer1M: 1.00, CachedInputPer1M: 0.20, OutputPer1M: 2.00, Source: xaiPricing, EffectiveFrom: verified},

		// Cohere. No cache-specific price is published for these rows, so cached
		// input is charged at the normal input rate.
		{Model: "command", NormalizedModel: "command", InputPer1M: 1.00, CachedInputPer1M: 1.00, OutputPer1M: 2.00, Source: coherePricing, EffectiveFrom: verified},
		{Model: "command-light", NormalizedModel: "command-light", InputPer1M: 0.30, CachedInputPer1M: 0.30, OutputPer1M: 0.60, Source: coherePricing, EffectiveFrom: verified},
		{Model: "command-r", NormalizedModel: "command-r", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: coherePricing, EffectiveFrom: verified},
		{Model: "command-r-plus", NormalizedModel: "command-r-plus", InputPer1M: 2.50, CachedInputPer1M: 2.50, OutputPer1M: 10.00, Source: coherePricing, EffectiveFrom: verified},
		{Model: "command-r-plus-04-2024", NormalizedModel: "command-r-plus-04-2024", InputPer1M: 3.00, CachedInputPer1M: 3.00, OutputPer1M: 15.00, Source: coherePricing, EffectiveFrom: verified},
		{Model: "command-r-plus-08-2024", NormalizedModel: "command-r-plus-08-2024", InputPer1M: 2.50, CachedInputPer1M: 2.50, OutputPer1M: 10.00, Source: coherePricing, EffectiveFrom: verified},
		{Model: "aya-expanse-8b", NormalizedModel: "aya-expanse-8b", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: coherePricing, EffectiveFrom: verified},
		{Model: "aya-expanse-32b", NormalizedModel: "aya-expanse-32b", InputPer1M: 0.50, CachedInputPer1M: 0.50, OutputPer1M: 1.50, Source: coherePricing, EffectiveFrom: verified},

		// Alibaba Cloud Model Studio / Qwen. Tiered rows use the lowest standard
		// listed tier unless the model only publishes a single global tier.
		{Model: "qwen3-max", NormalizedModel: "qwen3-max", InputPer1M: 1.65, CachedInputPer1M: 1.65, OutputPer1M: 4.951, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen-max", NormalizedModel: "qwen-max", InputPer1M: 1.60, CachedInputPer1M: 1.60, OutputPer1M: 6.40, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen3.6-plus", NormalizedModel: "qwen3.6-plus", InputPer1M: 0.276, CachedInputPer1M: 0.276, OutputPer1M: 1.651, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen-plus", NormalizedModel: "qwen-plus", InputPer1M: 0.40, CachedInputPer1M: 0.40, OutputPer1M: 4.00, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen-turbo", NormalizedModel: "qwen-turbo", InputPer1M: 0.05, CachedInputPer1M: 0.05, OutputPer1M: 0.50, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen3-coder-plus", NormalizedModel: "qwen3-coder-plus", InputPer1M: 0.574, CachedInputPer1M: 0.574, OutputPer1M: 2.294, Source: qwenPricing, EffectiveFrom: verified},
		{Model: "qwen3-coder-flash", NormalizedModel: "qwen3-coder-flash", InputPer1M: 0.144, CachedInputPer1M: 0.144, OutputPer1M: 0.574, Source: qwenPricing, EffectiveFrom: verified},
	}
	for _, rate := range rates {
		_, err := conn.ExecContext(ctx, `INSERT INTO pricing_models
			(model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0)
			ON CONFLICT(normalized_model) DO UPDATE SET
				model = excluded.model,
				input_per_1m = excluded.input_per_1m,
				cached_input_per_1m = excluded.cached_input_per_1m,
				output_per_1m = excluded.output_per_1m,
				source = excluded.source,
				effective_from = excluded.effective_from,
				is_custom = 0
			WHERE pricing_models.is_custom = 0`,
			rate.Model, rate.NormalizedModel, rate.InputPer1M, rate.CachedInputPer1M, rate.OutputPer1M, rate.Source, rate.EffectiveFrom.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return nil
}

func NormalizeModel(value string) string {
	return normalizeModelAlias(normalizeModelName(value))
}

func normalizeModelName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "models/")
	for _, prefix := range []string{
		"openai/",
		"anthropic/",
		"google/",
		"google-gemini/",
		"gemini/",
		"deepseek/",
		"mistral/",
		"xai/",
		"x-ai/",
		"tencent/",
		"tencent-hunyuan/",
		"hunyuan/",
		"cohere/",
		"qwen/",
		"dashscope/",
		"alibaba/",
	} {
		normalized = strings.TrimPrefix(normalized, prefix)
	}
	if strings.HasPrefix(normalized, "gpt") && len(normalized) > 3 && normalized[3] >= '0' && normalized[3] <= '9' {
		normalized = "gpt-" + normalized[3:]
	}
	return normalized
}

func normalizeModelAlias(normalized string) string {
	if alias, ok := modelAliases[normalized]; ok {
		return alias
	}
	return normalized
}

func normalizedModelCandidates(value string) []string {
	normalized := normalizeModelName(value)
	candidates := make([]string, 0, 4)
	seen := map[string]bool{}
	add := func(candidate string) {
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}

	for {
		add(normalizeModelAlias(normalized))
		index := strings.LastIndex(normalized, "-")
		if index <= 0 {
			break
		}
		normalized = normalized[:index]
	}
	return candidates
}

var modelAliases = map[string]string{
	"claude-4.5-haiku":    "claude-haiku-4.5",
	"claude-4.6-opus":     "claude-opus-4.6",
	"claude-4.6-sonnet":   "claude-sonnet-4.6",
	"claude-4.7-opus":     "claude-opus-4.7",
	"claude-opus-4-8":     "claude-opus-4.8",
	"claude-opus-4.6-1m":  "claude-opus-4.6",
	"claude-sonnet-4-6":   "claude-sonnet-4.6",
	"glm-5":               "glm-5.2",
	"glm-5.1":             "glm-5.2",
	"gpt-5.1-codex-mini":  "gpt-5-mini",
	"hy3":                 "hy3-preview",
	"hunyuan-hy3":         "hy3-preview",
	"hunyuan-hy3-preview": "hy3-preview",
}

func Compute(conn *sql.DB, usage model.Usage) (*float64, bool) {
	rate, ok := rateForUsage(conn, usage)
	return computeWithRate(usage, rate, ok)
}

func LoadCalculator(ctx context.Context, conn *sql.DB) (Calculator, error) {
	rows, err := conn.QueryContext(ctx, `SELECT model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from FROM pricing_models`)
	if err != nil {
		return Calculator{}, err
	}
	defer rows.Close()

	calc := Calculator{rates: map[string]Rate{}}
	for rows.Next() {
		var rate Rate
		var effective string
		if err := rows.Scan(&rate.Model, &rate.NormalizedModel, &rate.InputPer1M, &rate.CachedInputPer1M, &rate.OutputPer1M, &rate.Source, &effective); err != nil {
			return Calculator{}, err
		}
		rate.EffectiveFrom, _ = time.Parse(time.RFC3339Nano, effective)
		calc.rates[rate.NormalizedModel] = rate
	}
	return calc, rows.Err()
}

func UpsertCustom(ctx context.Context, conn *sql.DB, input model.PricingModelInput) (model.PricingModel, error) {
	rate, err := customRate(input)
	if err != nil {
		return model.PricingModel{}, err
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO pricing_models
		(model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(normalized_model) DO UPDATE SET
			model = excluded.model,
			input_per_1m = excluded.input_per_1m,
			cached_input_per_1m = excluded.cached_input_per_1m,
			output_per_1m = excluded.output_per_1m,
			source = excluded.source,
			effective_from = excluded.effective_from,
			is_custom = 1`,
		rate.Model,
		rate.NormalizedModel,
		rate.InputPer1M,
		rate.CachedInputPer1M,
		rate.OutputPer1M,
		rate.Source,
		rate.EffectiveFrom.Format(time.RFC3339Nano),
	); err != nil {
		return model.PricingModel{}, err
	}
	return get(ctx, conn, rate.NormalizedModel)
}

func customRate(input model.PricingModelInput) (Rate, error) {
	modelName := strings.TrimSpace(input.Model)
	if modelName == "" {
		return Rate{}, ErrInvalidRate
	}
	normalized := NormalizeModel(modelName)
	if normalized == "" || normalized == "unknown" {
		return Rate{}, ErrInvalidRate
	}
	for _, value := range []float64{input.InputPer1M, input.CachedInputPer1M, input.OutputPer1M} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return Rate{}, ErrInvalidRate
		}
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "Custom pricing"
	}
	return Rate{
		Model:            modelName,
		NormalizedModel:  normalized,
		InputPer1M:       input.InputPer1M,
		CachedInputPer1M: input.CachedInputPer1M,
		OutputPer1M:      input.OutputPer1M,
		Source:           source,
		EffectiveFrom:    time.Now().UTC(),
		IsCustom:         true,
	}, nil
}

func (c Calculator) Compute(usage model.Usage) (*float64, bool) {
	if !hasBillableUsage(usage) {
		return nil, false
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return nil, true
	}
	rate, ok := c.rateForModel(usage.Model)
	return computeWithRate(usage, rate, ok)
}

func (c Calculator) CacheSavings(usage model.Usage) *float64 {
	if !hasBillableUsage(usage) || usage.CachedInputTokens <= 0 {
		return nil
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return nil
	}
	rate, ok := c.rateForModel(usage.Model)
	if !ok || rate.InputPer1M <= rate.CachedInputPer1M {
		return nil
	}
	savings := float64(usage.CachedInputTokens) * (rate.InputPer1M - rate.CachedInputPer1M) / 1_000_000
	if savings <= 0 {
		return nil
	}
	return &savings
}

func (c Calculator) rateForModel(value string) (Rate, bool) {
	for _, candidate := range normalizedModelCandidates(value) {
		if rate, ok := c.rates[candidate]; ok {
			return rate, true
		}
	}
	return Rate{}, false
}

func rateForUsage(conn *sql.DB, usage model.Usage) (Rate, bool) {
	if !hasBillableUsage(usage) {
		return Rate{}, false
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return Rate{}, false
	}
	for _, candidate := range normalizedModelCandidates(usage.Model) {
		var rate Rate
		err := conn.QueryRow(`SELECT input_per_1m, cached_input_per_1m, output_per_1m FROM pricing_models WHERE normalized_model = ?`, candidate).
			Scan(&rate.InputPer1M, &rate.CachedInputPer1M, &rate.OutputPer1M)
		if err == nil {
			return rate, true
		}
		if err != sql.ErrNoRows {
			return Rate{}, false
		}
	}
	return Rate{}, false
}

func computeWithRate(usage model.Usage, rate Rate, hasRate bool) (*float64, bool) {
	if !hasBillableUsage(usage) {
		return nil, false
	}
	if usage.Model == "" || NormalizeModel(usage.Model) == "unknown" || !hasRate {
		return nil, true
	}
	uncachedInput, cachedInput := billableInputTokens(usage)
	outputTokens := billableOutputTokens(usage)
	cost := (float64(uncachedInput)*rate.InputPer1M + float64(cachedInput)*rate.CachedInputPer1M + float64(outputTokens)*rate.OutputPer1M) / 1_000_000
	return &cost, false
}

func billableInputTokens(usage model.Usage) (int64, int64) {
	inputTokens := usage.InputTokens
	if inputTokens < 0 {
		inputTokens = 0
	}
	cachedInputTokens := usage.CachedInputTokens
	if cachedInputTokens < 0 {
		cachedInputTokens = 0
	}
	if cachedInputTokens > inputTokens {
		return inputTokens, cachedInputTokens
	}
	return inputTokens - cachedInputTokens, cachedInputTokens
}

func billableOutputTokens(usage model.Usage) int64 {
	outputTokens := usage.OutputTokens
	if outputTokens < 0 {
		outputTokens = 0
	}
	reasoningTokens := usage.ReasoningOutputTokens
	if reasoningTokens <= 0 {
		return outputTokens
	}
	if usageReasoningOutputAppearsSeparate(usage, outputTokens, reasoningTokens) {
		return outputTokens + reasoningTokens
	}
	return outputTokens
}

func usageReasoningOutputAppearsSeparate(usage model.Usage, outputTokens, reasoningTokens int64) bool {
	if outputTokens <= 0 || reasoningTokens <= 0 || usage.TotalTokens <= 0 {
		return false
	}
	if reasoningTokens > outputTokens {
		return true
	}
	if !strings.Contains(NormalizeModel(usage.Model), "gemini") {
		return false
	}
	promptTokens := usage.InputTokens
	if usage.CachedInputTokens > usage.InputTokens {
		promptTokens += usage.CachedInputTokens
	}
	return usage.TotalTokens >= promptTokens+outputTokens+reasoningTokens
}

func hasBillableUsage(usage model.Usage) bool {
	return usage.InputTokens > 0 ||
		usage.CachedInputTokens > 0 ||
		usage.OutputTokens > 0 ||
		usage.ReasoningOutputTokens > 0 ||
		usage.TotalTokens > 0
}

func List(ctx context.Context, conn *sql.DB) ([]model.PricingModel, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id, model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom FROM pricing_models ORDER BY normalized_model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.PricingModel
	for rows.Next() {
		item, err := scanPricingModel(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func get(ctx context.Context, conn *sql.DB, normalizedModel string) (model.PricingModel, error) {
	row := conn.QueryRowContext(ctx, `SELECT id, model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom
		FROM pricing_models WHERE normalized_model = ?`, normalizedModel)
	return scanPricingModel(row)
}

func scanPricingModel(row interface{ Scan(dest ...any) error }) (model.PricingModel, error) {
	var item model.PricingModel
	var effective string
	var isCustom int
	if err := row.Scan(&item.ID, &item.Model, &item.NormalizedModel, &item.InputPer1M, &item.CachedInputPer1M, &item.OutputPer1M, &item.Source, &effective, &isCustom); err != nil {
		return model.PricingModel{}, err
	}
	item.EffectiveFrom, _ = time.Parse(time.RFC3339Nano, effective)
	item.IsCustom = isCustom != 0
	return item, nil
}
