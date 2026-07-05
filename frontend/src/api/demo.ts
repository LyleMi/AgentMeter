import type { DemoApi } from './demo/contracts'
import { agentResources } from './demo/agentResources'
import { auditFinding, auditSummary, filteredFindings, filteredToolCallRisks } from './demo/audit'
import { modelSignals } from './demo/modelSignals'
import { pricingModels, saveDemoPricingModel } from './demo/pricing'
import {
  deleteDemoPrompt,
  filteredPromptSuggestions,
  ignoreDemoPromptSuggestion,
  listSavedPrompts,
  recordDemoPromptCopy,
  saveDemoPrompt,
  unignoreDemoPromptSuggestion,
  updateDemoPrompt
} from './demo/prompts'
import { privacyStatus } from './demo/privacy'
import { filteredSessions, filteredToolCalls, sessionDetail, toolStatsFor } from './demo/sessions'
import { indexResult, settings } from './demo/settings'
import { breakdown, overview, tokenAnalytics } from './demo/usage'
import { clone, paginate } from './demo/utils'

function replaceAgentResources(next: typeof agentResources) {
  agentResources.agents = next.agents
  agentResources.skills = next.skills
  agentResources.mcpServers = next.mcpServers
  agentResources.memories = next.memories
  agentResources.warnings = next.warnings
  return agentResources
}

export const demoApi: DemoApi = {
  getSettings: async () => clone(settings()),
  getAgentResources: async () => clone(agentResources),
  setAgentSkillEnabled: async (input) => clone(replaceAgentResources({
    ...agentResources,
    skills: agentResources.skills.map((skill) => (
      skill.agentKind === input.agentKind && (skill.path === input.path || skill.name === input.name)
        ? { ...skill, enabled: input.enabled, status: input.enabled ? 'enabled' : 'disabled' }
        : skill
    ))
  })),
  setAgentMCPServerEnabled: async (input) => clone(replaceAgentResources({
    ...agentResources,
    mcpServers: agentResources.mcpServers.map((server) => (
      server.agentKind === input.agentKind && server.name === input.name
        ? { ...server, enabled: input.enabled, status: input.enabled ? 'configured' : 'disabled' }
        : server
    ))
  })),
  getAgentMemory: async (input) => {
    const memory = agentResources.memories.find((item) => item.agentKind === input.agentKind && item.path === input.path)
    if (!memory) throw new Error('Memory file not found')
    return clone(memory)
  },
  saveAgentMemory: async (input) => {
    const memory = agentResources.memories.find((item) => item.agentKind === input.agentKind && item.path === input.path)
    if (!memory) throw new Error('Memory file not found')
    memory.content = input.content
    memory.preview = input.content.trim().split(/\r?\n/).find(Boolean) || ''
    memory.sizeBytes = input.content.length
    memory.modifiedAt = new Date().toISOString()
    return clone(memory)
  },
  saveSourceSettings: async (sourceEntries) => clone(settings(sourceEntries)),
  getAgentPrivacy: async (target, _sourceKey) => clone(privacyStatus(target)),
  applyAgentPrivacyChanges: async (target, _changes, _sourceKey) => ({
    status: clone(privacyStatus(target)),
    changed: [],
    warnings: [
      'Static demo mode is read-only. Privacy changes were accepted for preview but not persisted.'
    ]
  }),
  applyAgentPrivacyProfile: async (target, _profile, _sourceKey) => ({
    status: clone(privacyStatus(target)),
    changed: [],
    warnings: [
      'Static demo mode is read-only. Privacy profile changes were accepted for preview but not persisted.'
    ]
  }),
  indexNow: async (rebuild = false) => clone(indexResult(rebuild)),
  getOverview: async (filters = {}) => clone(overview(filters)),
  getTokenAnalytics: async (filters = {}) => clone(tokenAnalytics(filters)),
  getModelSignals: async (filters = {}) => clone(modelSignals(filters)),
  getUsageBreakdown: async (filters) => clone(breakdown(filters)),
  getPromptSuggestions: async (filters = {}) => clone(filteredPromptSuggestions(filters)),
  listSavedPrompts: async () => clone(listSavedPrompts()),
  savePrompt: async (input) => clone(saveDemoPrompt(input)),
  updateSavedPrompt: async (id, input) => clone(updateDemoPrompt(id, input)),
  deleteSavedPrompt: async (id) => deleteDemoPrompt(id),
  recordPromptCopy: async (id) => clone(recordDemoPromptCopy(id)),
  ignorePromptSuggestion: async (suggestionKey) => ignoreDemoPromptSuggestion(suggestionKey),
  unignorePromptSuggestion: async (suggestionKey) => unignoreDemoPromptSuggestion(suggestionKey),
  listSessions: async (filters = {}) => clone(paginate(filteredSessions(filters), filters.limit, filters.offset)),
  getSessionDetail: async (id) => clone(sessionDetail(id)),
  getTools: async (filters = {}) => clone(toolStatsFor(filteredToolCalls({ agent: filters.agent }))),
  listToolCalls: async (filters = {}) => clone(paginate(filteredToolCalls(filters), filters.limit, filters.offset)),
  listToolCallRisks: async (filters = {}) => clone(filteredToolCallRisks(filters)),
  getAuditSummary: async (filters = {}) => clone(auditSummary(filters)),
  listAuditFindings: async (filters = {}) => clone(paginate(filteredFindings(filters), filters.limit, filters.offset)),
  getAuditFinding: async (id) => clone(auditFinding(id)),
  getPricingModels: async () => clone(pricingModels),
  savePricingModel: async (pricing) => clone(saveDemoPricingModel(pricing))
}
