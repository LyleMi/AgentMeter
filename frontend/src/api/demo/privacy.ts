import type {
  PrivacyConfigProfile,
  PrivacyConfigSetting,
  PrivacyConfigStatus,
  PrivacyTarget
} from '../types'

const profiles: PrivacyConfigProfile[] = [
  { id: 'default', title: 'Default', description: 'Leave vendor defaults in place.' },
  { id: 'recommended', title: 'Recommended', description: 'Disable telemetry while preserving local productivity features.' },
  { id: 'strict', title: 'Strict', description: 'Disable telemetry, network helpers, memory, and extended local retention.' }
]

type PrivacySettingSpec = {
  id: string
  group: string
  title: string
  key: string
  desiredValue: unknown
  strictValue: unknown
  currentValue: unknown
  configured: boolean
  status: string
  valueType?: PrivacyConfigSetting['valueType']
}

function privacySetting(spec: PrivacySettingSpec): PrivacyConfigSetting {
  const valueType = spec.valueType || 'bool'
  return {
    id: spec.id,
    group: spec.group,
    title: spec.title,
    description: `Demo status for ${spec.title.toLowerCase()}.`,
    key: spec.key,
    desiredValue: spec.desiredValue,
    strictValue: spec.strictValue,
    currentValue: spec.currentValue,
    valueType,
    configured: spec.configured,
    supportsUnset: true,
    status: spec.status,
    impact: `Controls ${spec.title.toLowerCase()} behavior for the selected agent.`,
    canApply: true,
    profileValues: [
      { profile: 'default', op: 'unset' },
      { profile: 'recommended', op: 'set', value: spec.desiredValue },
      { profile: 'strict', op: 'set', value: spec.strictValue }
    ]
  }
}

export function privacyStatus(target: PrivacyTarget): PrivacyConfigStatus {
  const targetName: Record<PrivacyTarget, string> = {
    codex: 'Codex',
    gemini: 'Gemini CLI',
    claude: 'Claude Code',
    codebuddy: 'CodeBuddy'
  }
  const settings = [
    privacySetting({ id: 'analytics.enabled', group: 'Telemetry', title: 'Analytics', key: 'analytics.enabled', desiredValue: false, strictValue: false, currentValue: false, configured: true, status: 'hardened' }),
    privacySetting({ id: 'telemetry.enabled', group: 'Telemetry', title: 'Telemetry export', key: 'telemetry.enabled', desiredValue: false, strictValue: false, currentValue: target === 'claude', configured: target !== 'claude', status: target === 'claude' ? 'attention' : 'hardened' }),
    privacySetting({ id: 'web_search', group: 'Network', title: 'Web search', key: 'web_search', desiredValue: false, strictValue: false, currentValue: false, configured: target === 'codex', status: target === 'codex' ? 'hardened' : 'implicit' }),
    privacySetting({ id: 'history.persistence', group: 'Local history', title: 'Conversation history', key: 'history.persistence', desiredValue: false, strictValue: false, currentValue: true, configured: false, status: 'implicit' }),
    privacySetting({ id: 'retention.days', group: 'Local retention', title: 'Retention days', key: 'retention.days', desiredValue: 14, strictValue: 7, currentValue: 14, configured: true, status: 'hardened', valueType: 'number' })
  ]
  const hardened = settings.filter((setting) => setting.status === 'hardened').length
  const attention = settings.filter((setting) => setting.status === 'attention').length
  const implicit = settings.filter((setting) => setting.status === 'implicit').length
  return {
    target,
    name: targetName[target],
    configPath: `C:\\Users\\demo\\.${target}\\${target === 'codex' ? 'config.toml' : 'settings.json'}`,
    exists: true,
    summary: {
      score: Math.round((hardened / settings.length) * 100),
      total: settings.length,
      hardened,
      attention,
      implicit
    },
    profiles,
    settings,
    warnings: ['Static demo mode is read-only. No local agent config will be changed.']
  }
}
