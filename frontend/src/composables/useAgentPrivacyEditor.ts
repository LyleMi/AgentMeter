import { ref, type Ref } from 'vue'
import type { PrivacyConfigChange, PrivacyConfigSetting, PrivacyConfigStatus, PrivacyConfigValueType } from '../api/types'
import {
  formatPrivacyConfigValue,
  normalizePrivacyConfigValue,
  privacyValueType,
  privacyValuesEqual,
  strictPrivacyValue
} from '../presentation/privacyConfig'

export type PrivacyConfigEditOp = 'set' | 'unset'

export interface PrivacyConfigEdit {
  id: string
  op: PrivacyConfigEditOp
  valueType: PrivacyConfigValueType
  boolValue: boolean
  stringValue: string
  numberValue: number
  arrayValue: string[]
}

type PrivacyConfigEditValueFields = Pick<
  PrivacyConfigEdit,
  'valueType' | 'boolValue' | 'stringValue' | 'numberValue' | 'arrayValue'
>

type PrivacyConfigEditMap = Record<string, PrivacyConfigEdit>

function editValueFields(value: unknown, type: PrivacyConfigValueType): PrivacyConfigEditValueFields {
  const normalized = normalizePrivacyConfigValue(value, type)
  return {
    valueType: type,
    boolValue: normalized === true,
    stringValue: typeof normalized === 'string' ? normalized : formatPrivacyConfigValue(normalized),
    numberValue: typeof normalized === 'number' ? normalized : Number(normalized) || 0,
    arrayValue: Array.isArray(normalized) ? normalized : []
  }
}

function createEdit(setting: PrivacyConfigSetting): PrivacyConfigEdit {
  const type = privacyValueType(setting)
  const baseValue = setting.configured ? setting.currentValue : strictPrivacyValue(setting)
  return {
    id: setting.id,
    op: setting.configured ? 'set' : 'unset',
    ...editValueFields(baseValue, type)
  }
}

function syncPrivacyEdits(edits: Ref<PrivacyConfigEditMap>, status: PrivacyConfigStatus | null) {
  const next: PrivacyConfigEditMap = {}
  for (const setting of status?.settings || []) {
    next[setting.id] = createEdit(setting)
  }
  edits.value = next
}

function editForSetting(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  if (!edits.value[setting.id]) edits.value[setting.id] = createEdit(setting)
  return edits.value[setting.id]
}

function editValueForSetting(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  const edit = editForSetting(edits, setting)
  if (edit.valueType === 'bool') return edit.boolValue
  if (edit.valueType === 'stringArray') return edit.arrayValue
  if (edit.valueType === 'number') return edit.numberValue
  return edit.stringValue
}

function markPrivacyEditSet(edits: Ref<PrivacyConfigEditMap>, id: string) {
  if (edits.value[id]) edits.value[id].op = 'set'
}

function useStrictPrivacyValue(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  const edit = editForSetting(edits, setting)
  const type = privacyValueType(setting)
  edit.op = 'set'
  Object.assign(edit, editValueFields(strictPrivacyValue(setting), type))
}

function unsetPrivacyEdit(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  editForSetting(edits, setting).op = 'unset'
}

function resetPrivacyEdit(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  edits.value[setting.id] = createEdit(setting)
}

function baselineOp(setting: PrivacyConfigSetting): PrivacyConfigEditOp {
  return setting.configured ? 'set' : 'unset'
}

function isPrivacyEditChanged(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting) {
  const edit = editForSetting(edits, setting)
  if (edit.op !== baselineOp(setting)) return true
  if (edit.op === 'unset') return false
  return !privacyValuesEqual(editValueForSetting(edits, setting), setting.currentValue, privacyValueType(setting))
}

function privacyChangeForSetting(edits: Ref<PrivacyConfigEditMap>, setting: PrivacyConfigSetting): PrivacyConfigChange {
  const edit = editForSetting(edits, setting)
  if (edit.op === 'unset') return { id: setting.id, op: 'unset' }
  return { id: setting.id, op: 'set', value: editValueForSetting(edits, setting) }
}

function canEditPrivacySetting(setting: PrivacyConfigSetting) {
  return setting.canApply !== false
}

export function useAgentPrivacyEditor() {
  const edits = ref<PrivacyConfigEditMap>({})

  return {
    edits,
    syncEdits: (status: PrivacyConfigStatus | null) => syncPrivacyEdits(edits, status),
    editFor: (setting: PrivacyConfigSetting) => editForSetting(edits, setting),
    editValue: (setting: PrivacyConfigSetting) => editValueForSetting(edits, setting),
    markEditSet: (id: string) => markPrivacyEditSet(edits, id),
    useStrict: (setting: PrivacyConfigSetting) => useStrictPrivacyValue(edits, setting),
    unsetEdit: (setting: PrivacyConfigSetting) => unsetPrivacyEdit(edits, setting),
    resetEdit: (setting: PrivacyConfigSetting) => resetPrivacyEdit(edits, setting),
    isEditChanged: (setting: PrivacyConfigSetting) => isPrivacyEditChanged(edits, setting),
    changeForSetting: (setting: PrivacyConfigSetting) => privacyChangeForSetting(edits, setting),
    canEdit: canEditPrivacySetting
  }
}
