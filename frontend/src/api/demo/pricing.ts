import type { PricingModel, PricingModelInput } from '../types'

export const pricingModels: PricingModel[] = [
  {
    id: 1,
    model: 'gpt-5-codex',
    normalizedModel: 'gpt-5-codex',
    inputPer1m: 1.25,
    cachedInputPer1m: 0.125,
    outputPer1m: 10,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z',
    isCustom: false
  },
  {
    id: 2,
    model: 'gemini-2.5-pro',
    normalizedModel: 'gemini-2.5-pro',
    inputPer1m: 1.25,
    cachedInputPer1m: 0.31,
    outputPer1m: 10,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z',
    isCustom: false
  },
  {
    id: 3,
    model: 'claude-sonnet-4',
    normalizedModel: 'claude-sonnet-4',
    inputPer1m: 3,
    cachedInputPer1m: 0.3,
    outputPer1m: 15,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z',
    isCustom: false
  }
]

export function costUsd(model: string, inputTokens: number, cachedInputTokens: number, outputTokens: number): number | undefined {
  const pricing = pricingModels.find((item) => item.normalizedModel === model)
  if (!pricing) return undefined
  return Number(
    (
      ((inputTokens - cachedInputTokens) * pricing.inputPer1m) / 1_000_000 +
      (cachedInputTokens * pricing.cachedInputPer1m) / 1_000_000 +
      (outputTokens * pricing.outputPer1m) / 1_000_000
    ).toFixed(4)
  )
}

function normalizePricingModelName(value: string) {
  return value.trim().toLowerCase().replace(/^models\//, '')
}

export function saveDemoPricingModel(input: PricingModelInput): PricingModel {
  const model = input.model.trim()
  if (!model) throw new Error('Model is required')
  if (input.inputPer1m < 0 || input.cachedInputPer1m < 0 || input.outputPer1m < 0) {
    throw new Error('Prices must be zero or greater')
  }
  const normalizedModel = normalizePricingModelName(model)
  const existingIndex = pricingModels.findIndex((item) => item.normalizedModel === normalizedModel)
  const saved: PricingModel = {
    id: existingIndex >= 0 ? pricingModels[existingIndex].id : Math.max(0, ...pricingModels.map((item) => item.id)) + 1,
    model,
    normalizedModel,
    inputPer1m: input.inputPer1m,
    cachedInputPer1m: input.cachedInputPer1m,
    outputPer1m: input.outputPer1m,
    source: input.source?.trim() || 'Custom pricing',
    effectiveFrom: new Date().toISOString(),
    isCustom: true
  }
  if (existingIndex >= 0) {
    pricingModels.splice(existingIndex, 1, saved)
  } else {
    pricingModels.push(saved)
  }
  return saved
}
