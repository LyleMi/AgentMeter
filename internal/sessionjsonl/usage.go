package sessionjsonl

import "github.com/LyleMi/AgentMeter/internal/model"

type usageSource struct {
	raw             map[string]any
	geminiMetadata  bool
	candidateOutput int64
}

type usageComponents struct {
	input              int64
	cached             int64
	cacheRead          int64
	output             int64
	reasoning          int64
	contextCompression int64
}

func readUsage(payload map[string]any, key string) model.Usage {
	info, _ := payload["info"].(map[string]any)
	if info == nil {
		return model.Usage{}
	}
	return usageFromValue(info[key])
}

func headlessUsage(raw rawRecord) model.Usage {
	for _, candidate := range usageCandidates(raw) {
		if usage := usageFromValue(candidate); hasUsage(usage) {
			return usage
		}
	}
	return model.Usage{}
}

func usageCandidates(raw rawRecord) []any {
	candidates := []any{raw.Usage}
	if raw.ProviderData != nil {
		candidates = append(candidates, raw.ProviderData["usage"], raw.ProviderData["rawUsage"])
	}
	for _, container := range []map[string]any{raw.Data, raw.Result, raw.Response, mapFromAny(raw.Message)} {
		if container != nil {
			candidates = append(candidates, container["usage"], container["usageMetadata"], container["usage_metadata"])
		}
	}
	return candidates
}

func usageFromValue(value any) model.Usage {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return model.Usage{}
	}
	source := normalizeUsageSource(raw)
	input, cached, cacheRead := usageInputComponents(source.raw)
	output, reasoning := usageOutputComponents(source)
	components := usageComponents{
		input:              input,
		cached:             cached,
		cacheRead:          cacheRead,
		output:             output,
		reasoning:          reasoning,
		contextCompression: contextCompressionTokensFromUsage(source.raw),
	}
	return model.Usage{
		InputTokens:              components.input,
		CachedInputTokens:        components.cached,
		OutputTokens:             components.output,
		ReasoningOutputTokens:    components.reasoning,
		ContextCompressionTokens: components.contextCompression,
		TotalTokens:              usageTotalTokens(source.raw, components),
	}
}

func normalizeUsageSource(raw map[string]any) usageSource {
	source := usageSource{raw: raw}
	if usageMetadata, ok := firstMap(raw, "usageMetadata", "usage_metadata"); ok {
		source.raw = usageMetadata
		source.geminiMetadata = true
	}
	source.candidateOutput = firstInt64(source.raw, "candidatesTokenCount", "candidates_token_count")
	if source.candidateOutput > 0 {
		source.geminiMetadata = true
	}
	return source
}

func usageInputComponents(raw map[string]any) (int64, int64, int64) {
	inputIncludesCached := false
	input := firstInt64(raw, "input_tokens", "input", "inputTokens", "promptTokenCount", "prompt_token_count")
	if input > 0 {
		input += firstInt64(raw, "cache_creation_input_tokens", "cache_write_input_tokens", "cacheCreationInputTokens", "cacheWriteInputTokens")
	} else {
		input = firstInt64(raw, "prompt_tokens", "promptTokens")
		inputIncludesCached = input > 0
	}
	cached := firstInt64(raw, "cached_input_tokens", "cache_read_input_tokens", "cached_tokens", "cachedInputTokens", "cacheReadInputTokens", "cachedTokens", "cachedContentTokenCount", "cached_content_token_count")
	cached += nestedInt64(raw["inputTokensDetails"], "cached_tokens", "cachedTokens")
	cached += nestedInt64(raw["input_tokens_details"], "cached_tokens", "cachedTokens")
	cached += nestedInt64(raw["prompt_tokens_details"], "cached_tokens", "cachedTokens")
	cacheRead := firstInt64(raw, "cache_read_input_tokens", "cacheReadInputTokens")
	if cacheRead == 0 && !inputIncludesCached {
		cacheRead = cached
	}
	return input, cached, cacheRead
}

func usageOutputComponents(source usageSource) (int64, int64) {
	output := firstInt64(source.raw, "output_tokens", "completion_tokens", "output", "outputTokens", "completionTokens")
	if source.candidateOutput > 0 {
		output = source.candidateOutput
	}
	reasoning := firstInt64(source.raw, "reasoning_output_tokens", "reasoning_tokens", "reasoningOutputTokens", "reasoningTokens", "completion_thinking_tokens", "thinking_tokens", "thinkingTokens", "thoughtsTokenCount", "thoughts_token_count")
	reasoning += nestedInt64(source.raw["outputTokensDetails"], "reasoning_tokens", "reasoningTokens")
	reasoning += nestedInt64(source.raw["output_tokens_details"], "reasoning_tokens", "reasoningTokens")
	reasoning += nestedInt64(source.raw["completion_tokens_details"], "reasoning_tokens", "reasoningTokens")
	if source.geminiMetadata && source.candidateOutput > 0 && reasoning > 0 {
		output += reasoning
	}
	return output, reasoning
}

func usageTotalTokens(raw map[string]any, components usageComponents) int64 {
	total := firstInt64(raw, "total_tokens", "totalTokens", "totalTokenCount", "total_token_count")
	observed := components.input + components.cached + components.output + components.reasoning + components.contextCompression
	if total <= 0 && observed > 0 {
		total = components.input + components.cacheRead + components.output
		if components.reasoning > components.output {
			total += components.reasoning
		}
		total += components.contextCompression
	}
	return total
}

func contextCompressionTokensFromUsage(raw map[string]any) int64 {
	keys := []string{
		"context_compression_tokens", "contextCompressionTokens",
		"context_compression_input_tokens", "contextCompressionInputTokens",
		"context_compaction_tokens", "contextCompactionTokens",
		"context_compaction_input_tokens", "contextCompactionInputTokens",
		"context_compressed_tokens", "contextCompressedTokens",
		"compaction_tokens", "compactionTokens", "compacted_tokens", "compactedTokens",
		"compression_tokens", "compressionTokens", "compressed_input_tokens", "compressedInputTokens",
	}
	total := firstInt64(raw, keys...)
	for _, value := range []any{raw["inputTokensDetails"], raw["input_tokens_details"], raw["prompt_tokens_details"], raw["contextTokensDetails"], raw["context_tokens_details"], raw["details"]} {
		total += nestedInt64(value, keys...)
	}
	return total
}

func contextCompressionTokensFromCompactMetadata(raw map[string]any) int64 {
	if raw == nil {
		return 0
	}
	preTokens := firstInt64(raw, "preTokens", "pre_tokens", "preTokenCount", "pre_token_count")
	postTokens := firstInt64(raw, "postTokens", "post_tokens", "postTokenCount", "post_token_count")
	return saturatingSubtract(preTokens, postTokens)
}

func firstInt64(payload map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value := int64Value(payload, key); value > 0 {
			return value
		}
	}
	return 0
}

func firstMap(payload map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		value, ok := payload[key].(map[string]any)
		if ok {
			return value, true
		}
	}
	return nil, false
}

func nestedInt64(value any, keys ...string) int64 {
	switch typed := value.(type) {
	case map[string]any:
		var total int64
		for _, key := range keys {
			total += int64Value(typed, key)
		}
		return total
	case []any:
		var total int64
		for _, item := range typed {
			total += nestedInt64(item, keys...)
		}
		return total
	default:
		return 0
	}
}

func hasUsage(usage model.Usage) bool {
	return usage.InputTokens > 0 || usage.CachedInputTokens > 0 || usage.OutputTokens > 0 || usage.ReasoningOutputTokens > 0 || usage.ContextCompressionTokens > 0 || usage.TotalTokens > 0
}

func subtractUsage(current model.Usage, previous *model.Usage) model.Usage {
	if previous == nil {
		return current
	}
	return model.Usage{
		InputTokens:              saturatingSubtract(current.InputTokens, previous.InputTokens),
		CachedInputTokens:        saturatingSubtract(current.CachedInputTokens, previous.CachedInputTokens),
		OutputTokens:             saturatingSubtract(current.OutputTokens, previous.OutputTokens),
		ReasoningOutputTokens:    saturatingSubtract(current.ReasoningOutputTokens, previous.ReasoningOutputTokens),
		ContextCompressionTokens: saturatingSubtract(current.ContextCompressionTokens, previous.ContextCompressionTokens),
		TotalTokens:              saturatingSubtract(current.TotalTokens, previous.TotalTokens),
	}
}

func addUsage(total *model.Usage, delta model.Usage) {
	if total.Source == "" || total.Source == "unknown" {
		total.Source = firstNonEmpty(delta.Source, "actual")
	}
	if total.Model == "" || total.Model == "unknown" {
		total.Model = delta.Model
	}
	total.InputTokens += delta.InputTokens
	total.CachedInputTokens += delta.CachedInputTokens
	total.OutputTokens += delta.OutputTokens
	total.ReasoningOutputTokens += delta.ReasoningOutputTokens
	total.ContextCompressionTokens += delta.ContextCompressionTokens
	total.TotalTokens += delta.TotalTokens
}

func saturatingSubtract(current, previous int64) int64 {
	if current <= previous {
		return 0
	}
	return current - previous
}
