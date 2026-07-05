import type {
  AgentResourceOverview,
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  IndexResult,
  ModelSignals,
  Overview,
  PricingModel,
  PricingModelInput,
  PromptSuggestion,
  PromptSuggestionFilters,
  PrivacyConfigApplyResult,
  PrivacyConfigChange,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget,
  SavedPrompt,
  SavedPromptInput,
  Session,
  SessionDetail,
  SessionFilters,
  Settings,
  SourceEntry,
  TokenAnalytics,
  ToolCall,
  ToolCallFilters,
  ToolFilters,
  ToolStat,
  UsageBreakdown,
  UsageBreakdownFilters,
  UsageScopeFilters
} from '../types'

export interface DemoSource {
  sourceId: number
  sourceKey: string
  sourceLabel: string
  sourceRootPath: string
  sourceSessionsPath: string
  agentKind: string
  agentName: string
}

export type DemoApi = {
  getSettings: () => Promise<Settings>
  getAgentResources: () => Promise<AgentResourceOverview>
  saveSourceSettings: (sourceEntries: SourceEntry[]) => Promise<Settings>
  getAgentPrivacy: (target: PrivacyTarget, sourceKey?: string) => Promise<PrivacyConfigStatus>
  applyAgentPrivacyChanges: (
    target: PrivacyTarget,
    changes: PrivacyConfigChange[],
    sourceKey?: string
  ) => Promise<PrivacyConfigApplyResult>
  applyAgentPrivacyProfile: (
    target: PrivacyTarget,
    profile: PrivacyProfileId,
    sourceKey?: string
  ) => Promise<PrivacyConfigApplyResult>
  indexNow: (rebuild?: boolean) => Promise<IndexResult>
  getOverview: (filters?: UsageScopeFilters) => Promise<Overview>
  getTokenAnalytics: (filters?: UsageScopeFilters) => Promise<TokenAnalytics>
  getModelSignals: (filters?: UsageScopeFilters) => Promise<ModelSignals>
  getUsageBreakdown: (filters: UsageBreakdownFilters) => Promise<UsageBreakdown>
  getPromptSuggestions: (filters?: PromptSuggestionFilters) => Promise<PromptSuggestion[]>
  listSavedPrompts: () => Promise<SavedPrompt[]>
  savePrompt: (input: SavedPromptInput) => Promise<SavedPrompt>
  updateSavedPrompt: (id: number, input: SavedPromptInput) => Promise<SavedPrompt>
  deleteSavedPrompt: (id: number) => Promise<{ ok: boolean }>
  recordPromptCopy: (id: number) => Promise<SavedPrompt>
  ignorePromptSuggestion: (suggestionKey: string) => Promise<{ ok: boolean }>
  unignorePromptSuggestion: (suggestionKey: string) => Promise<{ ok: boolean }>
  listSessions: (filters?: SessionFilters) => Promise<Session[]>
  getSessionDetail: (id: number) => Promise<SessionDetail>
  getTools: (filters?: ToolFilters) => Promise<ToolStat[]>
  listToolCalls: (filters?: ToolCallFilters) => Promise<ToolCall[]>
  getAuditSummary: (filters?: Pick<AuditFindingFilters, 'agent'>) => Promise<AuditSummary>
  listAuditFindings: (filters?: AuditFindingFilters) => Promise<AuditFinding[]>
  getAuditFinding: (id: number) => Promise<AuditFinding>
  getPricingModels: () => Promise<PricingModel[]>
  savePricingModel: (pricing: PricingModelInput) => Promise<PricingModel>
}
