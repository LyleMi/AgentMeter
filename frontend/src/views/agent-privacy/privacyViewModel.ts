import { computed, type Ref } from 'vue'
import type {
  PrivacyConfigApplyResult,
  PrivacyConfigSetting,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget
} from '../../api/types'
import { privacySettingMatchesStrict } from '../../presentation/privacyConfig'
import type {
  PrivacyProfileOption,
  PrivacySettingGroup,
  PrivacyTargetOption,
  PrivacyTranslate
} from '../../presentation/privacyUi'

const profileIds: PrivacyProfileId[] = ['default', 'recommended', 'strict']

interface AgentPrivacyViewModelOptions {
  t: PrivacyTranslate
  selectedTarget: Ref<PrivacyTarget>
  privacyStatus: Ref<PrivacyConfigStatus | null>
  lastApply: Ref<PrivacyConfigApplyResult | null>
  canEdit: (setting: PrivacyConfigSetting) => boolean
  isEditChanged: (setting: PrivacyConfigSetting) => boolean
  localizedSettingGroup: (setting: PrivacyConfigSetting) => string
  profileTitle: (profile: PrivacyProfileId) => string
}

function privacyTargetOptions(t: PrivacyTranslate): PrivacyTargetOption[] {
  return [
    { label: t('privacy.target.codex'), value: 'codex' },
    { label: t('privacy.target.gemini'), value: 'gemini' },
    { label: t('privacy.target.claude'), value: 'claude' },
    { label: t('privacy.target.codebuddy'), value: 'codebuddy' }
  ]
}

function privacyProfileOptions(
  t: PrivacyTranslate,
  profileTitle: (profile: PrivacyProfileId) => string
): PrivacyProfileOption[] {
  return profileIds.map((profile) => ({
    id: profile,
    title: profileTitle(profile),
    description: t(`privacy.profile.${profile}.description`)
  }))
}

function fallbackSummary() {
  return {
    score: 0,
    total: 0,
    hardened: 0,
    attention: 0,
    implicit: 0
  }
}

function uniquePrivacyWarnings(
  privacyStatus: PrivacyConfigStatus | null,
  lastApply: PrivacyConfigApplyResult | null
) {
  const values = [...(privacyStatus?.warnings || []), ...(lastApply?.warnings || [])]
  return [...new Set(values.filter(Boolean))]
}

function privacyMetricCounts(
  settings: PrivacyConfigSetting[],
  totalFallback: number,
  changedSettings: PrivacyConfigSetting[]
) {
  const total = settings.length || totalFallback
  const strictConfigured = settings.filter(privacySettingMatchesStrict).length
  const defaultSafe = settings.filter((setting) => !setting.configured && setting.status === 'implicit').length
  const customConfigured = settings.filter((setting) => setting.configured && !privacySettingMatchesStrict(setting)).length
  const missingRequired = settings.filter((setting) => !setting.configured && setting.status === 'attention').length
  const unsavedChanges = changedSettings.length
  return { total, strictConfigured, defaultSafe, customConfigured, missingRequired, unsavedChanges }
}

function privacyStatusState(
  t: PrivacyTranslate,
  hasStatus: boolean,
  missingRequired: number
) {
  if (!hasStatus) return { color: 'default', label: t('privacy.status.noStatus') }
  if (missingRequired > 0) return { color: 'warning', label: t('privacy.status.needsChange') }
  return { color: 'success', label: t('privacy.status.ready') }
}

function groupedPrivacySettings(
  settings: PrivacyConfigSetting[],
  localizedSettingGroup: (setting: PrivacyConfigSetting) => string
): PrivacySettingGroup[] {
  const groups = new Map<string, PrivacyConfigSetting[]>()
  for (const setting of settings) {
    const group = localizedSettingGroup(setting)
    groups.set(group, [...(groups.get(group) || []), setting])
  }
  return [...groups.entries()].map(([name, items]) => ({ name, items }))
}

export function useAgentPrivacyViewModel(options: AgentPrivacyViewModelOptions) {
  const { t, selectedTarget, privacyStatus, lastApply, canEdit, isEditChanged, localizedSettingGroup, profileTitle } =
    options

  const targetOptions = computed<PrivacyTargetOption[]>(() => privacyTargetOptions(t))
  const profileOptions = computed<PrivacyProfileOption[]>(() => privacyProfileOptions(t, profileTitle))
  const targetLabel = computed(() => {
    if (privacyStatus.value?.name) return privacyStatus.value.name
    return targetOptions.value.find((option) => option.value === selectedTarget.value)?.label || selectedTarget.value
  })
  const targetFile = computed(() => (selectedTarget.value === 'codex' ? 'config.toml' : 'settings.json'))
  const summary = computed(() => privacyStatus.value?.summary || fallbackSummary())
  const settings = computed(() => privacyStatus.value?.settings || [])
  const changedSettings = computed(() => settings.value.filter((setting) => canEdit(setting) && isEditChanged(setting)))
  const kickerText = computed(() => t('privacy.kicker', { target: targetLabel.value, file: targetFile.value }))
  const warningList = computed(() => uniquePrivacyWarnings(privacyStatus.value, lastApply.value))
  const metricCounts = computed(() => privacyMetricCounts(settings.value, summary.value.total, changedSettings.value))
  const statusState = computed(() =>
    privacyStatusState(t, Boolean(privacyStatus.value), metricCounts.value.missingRequired)
  )
  const groupedSettings = computed<PrivacySettingGroup[]>(() =>
    groupedPrivacySettings(settings.value, localizedSettingGroup)
  )

  return {
    targetOptions,
    profileOptions,
    targetLabel,
    settings,
    changedSettings,
    kickerText,
    statusState,
    warningList,
    metricCounts,
    groupedSettings
  }
}
