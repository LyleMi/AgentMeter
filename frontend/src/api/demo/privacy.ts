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

function privacySetting(
  id: string,
  group: string,
  title: string,
  key: string,
  desiredValue: unknown,
  strictValue: unknown,
  currentValue: unknown,
  configured: boolean,
  status: string,
  valueType: PrivacyConfigSetting['valueType'] = 'bool'
): PrivacyConfigSetting {
  return {
    id,
    group,
    title,
    description: `Demo status for ${title.toLowerCase()}.`,
    key,
    desiredValue,
    strictValue,
    currentValue,
    valueType,
    configured,
    supportsUnset: true,
    status,
    impact: `Controls ${title.toLowerCase()} behavior for the selected agent.`,
    canApply: true,
    profileValues: [
      { profile: 'default', op: 'unset' },
      { profile: 'recommended', op: 'set', value: desiredValue },
      { profile: 'strict', op: 'set', value: strictValue }
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
    privacySetting('analytics.enabled', 'Telemetry', 'Analytics', 'analytics.enabled', false, false, false, true, 'hardened'),
    privacySetting('telemetry.enabled', 'Telemetry', 'Telemetry export', 'telemetry.enabled', false, false, target === 'claude', target !== 'claude', target === 'claude' ? 'attention' : 'hardened'),
    privacySetting('web_search', 'Network', 'Web search', 'web_search', false, false, false, target === 'codex', target === 'codex' ? 'hardened' : 'implicit'),
    privacySetting('history.persistence', 'Local history', 'Conversation history', 'history.persistence', false, false, true, false, 'implicit'),
    privacySetting('retention.days', 'Local retention', 'Retention days', 'retention.days', 14, 7, 14, true, 'hardened', 'number')
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
