import type {
  ModelSignalProjectHotspot,
  ModelSignalsProjectMetric
} from '../../api'

export type ProjectMetricRow = ModelSignalsProjectMetric | ModelSignalProjectHotspot

export interface NormalizedAnomalySession {
  id: number
  sessionKey?: string
  codexSessionId?: string
  startedAt?: string
  projectPath?: string
  rawSourcePath?: string
  agentKind?: string
  agentName?: string
  sourceId?: number
  sourceKey?: string
  sourceLabel?: string
  sourceRootPath?: string
  sourceSessionsPath?: string
  model?: string
  totalTokens: number
  outputExpansionRate: number
  reasoningOverheadRate: number
  cacheMissRate: number
  modelThroughputTokensPerSecond: number
  failedToolCalls: number
  modelDurationMs: number
  score: number
  reasons: string[]
}
