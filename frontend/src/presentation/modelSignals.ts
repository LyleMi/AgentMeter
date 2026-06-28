import { formatPercent as formatSharedPercent } from '../api'

export function formatModelSignalPercent(value?: number) {
  const numeric = Number(value)
  return formatSharedPercent(value, {
    lessThanOne: true,
    maximumFractionDigits: Number.isFinite(numeric) && numeric >= 0.1 ? 0 : 1
  })
}

export function formatModelSignalRate(value?: number, digits = 0) {
  if (!Number.isFinite(value)) return '0'
  return (value || 0).toLocaleString(undefined, { maximumFractionDigits: digits })
}
