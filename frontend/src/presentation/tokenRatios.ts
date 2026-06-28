export interface TokenRatioInput {
  inputTokens?: number | null
  cachedInputTokens?: number | null
  outputTokens?: number | null
  reasoningOutputTokens?: number | null
}

export interface TokenRatioShares {
  input: number
  cachedInput: number
  output: number
  reasoningOutput: number
}

export function tokenRatioShares(input: TokenRatioInput): TokenRatioShares {
  const inputTokens = positiveNumber(input.inputTokens)
  const cachedInputTokens = positiveNumber(input.cachedInputTokens)
  const outputTokens = positiveNumber(input.outputTokens)
  const reasoningOutputTokens = positiveNumber(input.reasoningOutputTokens)
  const mainTotal = inputTokens + outputTokens

  return {
    input: mainTotal > 0 ? inputTokens / mainTotal : 0,
    cachedInput: cachedInputRatio(inputTokens, cachedInputTokens),
    output: mainTotal > 0 ? outputTokens / mainTotal : 0,
    reasoningOutput: outputTokens > 0 ? clamp01(reasoningOutputTokens / outputTokens) : 0
  }
}

export function cachedInputRatio(inputTokens?: number | null, cachedInputTokens?: number | null) {
  const input = positiveNumber(inputTokens)
  const cached = positiveNumber(cachedInputTokens)
  const denominator = cacheInputDenominator(input, cached)
  if (denominator <= 0 || cached <= 0) return 0
  return clamp01(cached / denominator)
}

function cacheInputDenominator(inputTokens: number, cachedInputTokens: number) {
  if (inputTokens <= 0) {
    return cachedInputTokens > 0 ? cachedInputTokens : 0
  }
  if (cachedInputTokens > inputTokens) {
    return inputTokens + cachedInputTokens
  }
  return inputTokens
}

function positiveNumber(value?: number | null) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, value || 0)
}

function clamp01(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(1, value))
}
