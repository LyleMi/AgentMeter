import { computed, type Ref } from 'vue'
import type {
  PrivacyConfigApplyResult,
  PrivacyConfigSetting,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget
} from '../../api/types'
import { privacyValueType, privacyValuesEqual, strictPrivacyValue } from '../../presentation/privacyConfig'

type Translate = (key: string, params?: Record<string, string>) => string

const profileIds: PrivacyProfileId[] = ['default', 'recommended', 'strict']

export function useAgentPrivacyViewModel(options: {
  t: Translate
  selectedTarget: Ref<PrivacyTarget>
  privacyStatus: Ref<PrivacyConfigStatus | null>
  lastApply: Ref<PrivacyConfigApplyResult | null>
  canEdit: (setting: PrivacyConfigSetting) => boolean
  isEditChanged: (setting: PrivacyConfigSetting) => boolean
  localizedSettingGroup: (setting: PrivacyConfigSetting) => string
  profileTitle: (profile: PrivacyProfileId) => string
}) {
  const { t, selectedTarget, privacyStatus, lastApply, canEdit, isEditChanged, localizedSettingGroup, profileTitle } =
    options

  const targetOptions = computed<{ label: string; value: PrivacyTarget }[]>(() => [
    { label: t('privacy.target.codex'), value: 'codex' },
    { label: t('privacy.target.gemini'), value: 'gemini' },
    { label: t('privacy.target.claude'), value: 'claude' },
    { label: t('privacy.target.codebuddy'), value: 'codebuddy' }
  ])
  const profileOptions = computed(() =>
    profileIds.map((profile) => ({
      id: profile,
      title: profileTitle(profile),
      description: t(`privacy.profile.${profile}.description`)
    }))
  )
  const targetLabel = computed(() => {
    if (privacyStatus.value?.name) return privacyStatus.value.name
    return targetOptions.value.find((option) => option.value === selectedTarget.value)?.label || selectedTarget.value
  })
  const targetFile = computed(() => (selectedTarget.value === 'codex' ? 'config.toml' : 'settings.json'))
  const summary = computed(
    () =>
      privacyStatus.value?.summary || {
        score: 0,
        total: 0,
        hardened: 0,
        attention: 0,
        implicit: 0
      }
  )
  const settings = computed(() => privacyStatus.value?.settings || [])
  const changedSettings = computed(() => settings.value.filter((setting) => canEdit(setting) && isEditChanged(setting)))
  const kickerText = computed(() => t('privacy.kicker', { target: targetLabel.value, file: targetFile.value }))
  const warningList = computed(() => {
    const values = [...(privacyStatus.value?.warnings || []), ...(lastApply.value?.warnings || [])]
    return [...new Set(values.filter(Boolean))]
  })
  const metricCounts = computed(() => {
    const total = settings.value.length || summary.value.total
    const strictConfigured = settings.value.filter(
      (setting) =>
        setting.configured &&
        privacyValuesEqual(setting.currentValue, strictPrivacyValue(setting), privacyValueType(setting))
    ).length
    const defaultSafe = settings.value.filter((setting) => !setting.configured && setting.status === 'implicit').length
    const customConfigured = settings.value.filter(
      (setting) =>
        setting.configured &&
        !privacyValuesEqual(setting.currentValue, strictPrivacyValue(setting), privacyValueType(setting))
    ).length
    const missingRequired = settings.value.filter((setting) => !setting.configured && setting.status === 'attention').length
    const unsavedChanges = changedSettings.value.length
    return { total, strictConfigured, defaultSafe, customConfigured, missingRequired, unsavedChanges }
  })
  const statusState = computed(() => {
    if (!privacyStatus.value) return { color: 'default', label: t('privacy.status.noStatus') }
    if (metricCounts.value.missingRequired > 0) {
      return { color: 'warning', label: t('privacy.status.needsChange') }
    }
    return { color: 'success', label: t('privacy.status.ready') }
  })
  const groupedSettings = computed(() => {
    const groups = new Map<string, PrivacyConfigSetting[]>()
    for (const setting of settings.value) {
      const group = localizedSettingGroup(setting)
      groups.set(group, [...(groups.get(group) || []), setting])
    }
    return [...groups.entries()].map(([name, items]) => ({ name, items }))
  })

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
