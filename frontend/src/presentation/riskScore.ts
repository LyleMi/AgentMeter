interface ThresholdRiskInput {
  value: number
  warning: number
  critical: number
}

interface RangeRiskInput {
  value: number
  start: number
  span: number
}

export function clampRiskScore(value: number): number {
  if (!Number.isFinite(value)) return 0
  if (value < 0) return 0
  if (value > 1) return 1
  return value
}

export function thresholdRiskScore(input: ThresholdRiskInput): number {
  if (input.value <= input.warning || input.warning >= input.critical) return 0
  if (input.value >= input.critical) return 1
  return clampRiskScore((input.value - input.warning) / (input.critical - input.warning))
}

export function inverseThresholdRiskScore(input: ThresholdRiskInput): number {
  if (input.value <= 0 || input.warning <= input.critical) return 0
  if (input.value >= input.warning) return 0
  if (input.value <= input.critical) return 1
  return clampRiskScore((input.warning - input.value) / (input.warning - input.critical))
}

export function rangeRiskScore(input: RangeRiskInput): number {
  if (input.value <= input.start || input.span <= 0) return 0
  return clampRiskScore((input.value - input.start) / input.span)
}
