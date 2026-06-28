import type { Ref } from 'vue'
import type { PrivacyConfigSetting, PrivacyProfileId, PrivacyTarget } from '../../api/types'
import type { PrivacyTranslate } from '../../presentation/privacyUi'
import {
  formatPrivacyConfigValue,
  privacyProfileValue,
  privacySettingMatchesStrict,
  strictPrivacyValue
} from '../../presentation/privacyConfig'

type SettingTextField = 'title' | 'description' | 'impact'

const groupMessageKeys: Record<string, string> = {
  Telemetry: 'privacy.settingGroup.telemetry',
  Network: 'privacy.settingGroup.network',
  'Local history': 'privacy.settingGroup.localHistory',
  Memory: 'privacy.settingGroup.memory',
  Environment: 'privacy.settingGroup.environment',
  Usage: 'privacy.settingGroup.usage',
  'Local retention': 'privacy.settingGroup.localRetention',
  Approval: 'privacy.settingGroup.approval',
  Extensions: 'privacy.settingGroup.extensions',
  Browser: 'privacy.settingGroup.browser',
  Voice: 'privacy.settingGroup.voice'
}

const settingMessageBases: Partial<Record<PrivacyTarget, Record<string, string>>> = {
  codex: {
    'analytics.enabled': 'privacy.setting.codex.analytics.enabled',
    'otel.exporter': 'privacy.setting.codex.otel.exporter',
    'otel.trace_exporter': 'privacy.setting.codex.otel.trace_exporter',
    'otel.metrics_exporter': 'privacy.setting.codex.otel.metrics_exporter',
    'otel.log_user_prompt': 'privacy.setting.codex.otel.log_user_prompt',
    web_search: 'privacy.setting.codex.web_search',
    'history.persistence': 'privacy.setting.codex.history.persistence',
    'features.memories': 'privacy.setting.codex.features.memories',
    'memories.generate_memories': 'privacy.setting.codex.memories.generate_memories',
    'memories.use_memories': 'privacy.setting.codex.memories.use_memories',
    'memories.disable_on_external_context': 'privacy.setting.codex.memories.disable_on_external_context',
    'sandbox_workspace_write.network_access': 'privacy.setting.codex.sandbox_workspace_write.network_access',
    'shell_environment_policy.inherit': 'privacy.setting.codex.shell_environment_policy.inherit',
    'shell_environment_policy.ignore_default_excludes': 'privacy.setting.codex.shell_environment_policy.ignore_default_excludes'
  },
  gemini: {
    'privacy.usageStatisticsEnabled': 'privacy.setting.gemini.privacy.usageStatisticsEnabled',
    'telemetry.enabled': 'privacy.setting.gemini.telemetry.enabled',
    'telemetry.traces': 'privacy.setting.gemini.telemetry.traces',
    'telemetry.logPrompts': 'privacy.setting.gemini.telemetry.logPrompts',
    'general.logRagSnippets': 'privacy.setting.gemini.general.logRagSnippets',
    'general.checkpointing.enabled': 'privacy.setting.gemini.general.checkpointing.enabled',
    'general.sessionRetention.enabled': 'privacy.setting.gemini.general.sessionRetention.enabled',
    'general.sessionRetention.maxAge': 'privacy.setting.gemini.general.sessionRetention.maxAge',
    'tools.sandboxNetworkAccess': 'privacy.setting.gemini.tools.sandboxNetworkAccess',
    'tools.exclude.web': 'privacy.setting.gemini.tools.exclude.web',
    'experimental.directWebFetch': 'privacy.setting.gemini.experimental.directWebFetch',
    'advanced.ignoreLocalEnv': 'privacy.setting.gemini.advanced.ignoreLocalEnv',
    'security.environmentVariableRedaction.enabled':
      'privacy.setting.gemini.security.environmentVariableRedaction.enabled',
    'security.disableYoloMode': 'privacy.setting.gemini.security.disableYoloMode',
    'security.disableAlwaysAllow': 'privacy.setting.gemini.security.disableAlwaysAllow',
    'security.enablePermanentToolApproval': 'privacy.setting.gemini.security.enablePermanentToolApproval',
    'security.blockGitExtensions': 'privacy.setting.gemini.security.blockGitExtensions',
    'agents.browser.confirmSensitiveActions': 'privacy.setting.gemini.agents.browser.confirmSensitiveActions',
    'agents.browser.blockFileUploads': 'privacy.setting.gemini.agents.browser.blockFileUploads',
    'experimental.voiceMode': 'privacy.setting.gemini.experimental.voiceMode',
    'experimental.autoMemory': 'privacy.setting.gemini.experimental.autoMemory',
    'context.loadMemoryFromIncludeDirectories':
      'privacy.setting.gemini.context.loadMemoryFromIncludeDirectories',
    'skills.enabled': 'privacy.setting.gemini.skills.enabled'
  }
}

export function useAgentPrivacyDisplay(options: {
  t: PrivacyTranslate
  locale: Ref<string>
  activeTarget: () => PrivacyTarget | undefined
}) {
  const { t, locale, activeTarget } = options

  function localizedMessage(key: string | undefined, fallback: string) {
    if (!key || locale.value === 'en') return fallback
    const localized = t(key)
    return localized === key ? fallback : localized
  }

  function localizedSettingGroup(setting: PrivacyConfigSetting) {
    if (!setting.group) return t('privacy.group.default')
    return localizedMessage(groupMessageKeys[setting.group], setting.group)
  }

  function settingMessageKey(setting: PrivacyConfigSetting, field: SettingTextField) {
    const target = activeTarget()
    const base = target ? settingMessageBases[target]?.[setting.id] : undefined
    return base ? `${base}.${field}` : undefined
  }

  function localizedSettingText(setting: PrivacyConfigSetting, field: SettingTextField, fallback: string) {
    return localizedMessage(settingMessageKey(setting, field), fallback)
  }

  function formatConfigValue(value: unknown) {
    return formatPrivacyConfigValue(value, t('privacy.value.unset'))
  }

  function profileTitle(profile: PrivacyProfileId) {
    return t(`privacy.profile.${profile}.title`)
  }

  function profileBehaviorText(setting: PrivacyConfigSetting, profile: PrivacyProfileId) {
    const profileValue = privacyProfileValue(setting, profile)
    if (!profileValue) {
      if (profile === 'default') return t('privacy.value.profileUnset')
      if (profile === 'strict') return formatConfigValue(strictPrivacyValue(setting))
      return t('privacy.value.profileUnavailable')
    }
    if (profileValue.op === 'set') return formatConfigValue(profileValue.value)
    if (profileValue.op === 'unset') return t('privacy.value.profileUnset')
    return t('privacy.value.profileNone')
  }

  function settingState(setting: PrivacyConfigSetting) {
    if (privacySettingMatchesStrict(setting)) {
      return { color: 'success', label: t('privacy.status.hardened') }
    }
    if (setting.configured) return { color: 'processing', label: t('privacy.meta.customConfigured') }
    if (setting.status === 'implicit') return { color: 'default', label: t('privacy.status.implicit') }
    return { color: 'warning', label: t('privacy.value.notConfigured') }
  }

  return {
    formatConfigValue,
    profileTitle,
    profileBehaviorText,
    settingState,
    localizedSettingGroup,
    localizedSettingTitle: (setting: PrivacyConfigSetting) =>
      localizedSettingText(setting, 'title', setting.title),
    localizedSettingDescription: (setting: PrivacyConfigSetting) =>
      localizedSettingText(setting, 'description', setting.description),
    localizedSettingImpact: (setting: PrivacyConfigSetting) =>
      localizedSettingText(setting, 'impact', setting.impact)
  }
}
