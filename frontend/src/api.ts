export { api, isStaticDemo } from './api/client'
export type * from './api/types'
export {
  formatCost,
  formatBytes,
  formatDateTime,
  formatDisplayCost,
  formatDisplayNumber,
  formatDuration,
  formatPercent,
  formatNumber,
  projectDisplay,
  projectName,
  sessionDisplay,
  sessionFullLabel,
  sessionLabel,
  shortPath
} from './presentation/formatters'
export type { CollapsedText, DisplayNumber } from './presentation/formatters'
