import type { PrivacyConfigSetting, PrivacyConfigValueType, PrivacyProfileId } from '../api/types'

export function privacyValueType(setting: PrivacyConfigSetting): PrivacyConfigValueType {
  if (setting.valueType) return setting.valueType
  const sample = strictPrivacyValue(setting) ?? setting.currentValue
  if (typeof sample === 'boolean') return 'bool'
  if (Array.isArray(sample)) return 'stringArray'
  if (typeof sample === 'number') return 'number'
  return 'string'
}

export function strictPrivacyValue(setting: PrivacyConfigSetting) {
  const strictProfileValue = privacyProfileValue(setting, 'strict')
  if (strictProfileValue?.op === 'set') return strictProfileValue.value
  return setting.strictValue !== undefined ? setting.strictValue : setting.desiredValue
}

export function privacyProfileValue(setting: PrivacyConfigSetting, profile: PrivacyProfileId) {
  return setting.profileValues?.find((profileValue) => profileValue.profile === profile)
}

export function normalizePrivacyConfigValue(value: unknown, type: PrivacyConfigValueType): unknown {
  if (type === 'bool') {
    if (typeof value === 'string') return value.toLowerCase() === 'true'
    return value === true
  }
  if (type === 'stringArray') {
    if (!Array.isArray(value)) return value === undefined || value === null || value === '' ? [] : [String(value)]
    return value.map((item) => String(item))
  }
  if (type === 'number') {
    if (typeof value === 'number' && Number.isFinite(value)) return value
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : formatPrivacyConfigValue(value)
}

export function privacyValuesEqual(left: unknown, right: unknown, type: PrivacyConfigValueType) {
  return JSON.stringify(normalizePrivacyConfigValue(left, type)) === JSON.stringify(normalizePrivacyConfigValue(right, type))
}

export function formatPrivacyConfigValue(value: unknown, unsetLabel = 'unset') {
  if (value === undefined || value === null) return unsetLabel
  if (value === '') return '""'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)

  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export function privacyValueTypeLabel(type: PrivacyConfigValueType) {
  if (type === 'bool') return 'bool'
  if (type === 'stringArray') return 'string[]'
  if (type === 'number') return 'number'
  return 'string'
}
