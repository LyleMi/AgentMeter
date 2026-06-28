import type { PrivacyConfigSetting, PrivacyProfileId, PrivacyTarget } from '../api/types'

export type PrivacyTranslate = (key: string, params?: Record<string, string>) => string

export interface PrivacyTargetOption {
  label: string
  value: PrivacyTarget
}

export interface PrivacyStatusState {
  color: string
  label: string
}

export interface PrivacyProfileOption {
  id: PrivacyProfileId
  title: string
  description: string
}

export interface PrivacyMetricCounts {
  total: number
  strictConfigured: number
  defaultSafe: number
  customConfigured: number
  missingRequired: number
  unsavedChanges: number
}

export interface PrivacySettingGroup {
  name: string
  items: PrivacyConfigSetting[]
}
