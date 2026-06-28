import { ref } from 'vue'
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

export function useAgentPrivacyEditor() {
  const edits = ref<Record<string, PrivacyConfigEdit>>({})

  function createEdit(setting: PrivacyConfigSetting): PrivacyConfigEdit {
    const type = privacyValueType(setting)
    const baseValue = setting.configured ? setting.currentValue : strictPrivacyValue(setting)
    return {
      id: setting.id,
      op: setting.configured ? 'set' : 'unset',
      ...editValueFields(baseValue, type)
    }
  }

  function syncEdits(status: PrivacyConfigStatus | null) {
    const next: Record<string, PrivacyConfigEdit> = {}
    for (const setting of status?.settings || []) {
      next[setting.id] = createEdit(setting)
    }
    edits.value = next
  }

  function editFor(setting: PrivacyConfigSetting) {
    if (!edits.value[setting.id]) edits.value[setting.id] = createEdit(setting)
    return edits.value[setting.id]
  }

  function editValue(setting: PrivacyConfigSetting) {
    const edit = editFor(setting)
    if (edit.valueType === 'bool') return edit.boolValue
    if (edit.valueType === 'stringArray') return edit.arrayValue
    if (edit.valueType === 'number') return edit.numberValue
    return edit.stringValue
  }

  function markEditSet(id: string) {
    if (edits.value[id]) edits.value[id].op = 'set'
  }

  function useStrict(setting: PrivacyConfigSetting) {
    const edit = editFor(setting)
    const type = privacyValueType(setting)
    edit.op = 'set'
    Object.assign(edit, editValueFields(strictPrivacyValue(setting), type))
  }

  function unsetEdit(setting: PrivacyConfigSetting) {
    editFor(setting).op = 'unset'
  }

  function resetEdit(setting: PrivacyConfigSetting) {
    edits.value[setting.id] = createEdit(setting)
  }

  function baselineOp(setting: PrivacyConfigSetting): PrivacyConfigEditOp {
    return setting.configured ? 'set' : 'unset'
  }

  function isEditChanged(setting: PrivacyConfigSetting) {
    const edit = editFor(setting)
    if (edit.op !== baselineOp(setting)) return true
    if (edit.op === 'unset') return false
    return !privacyValuesEqual(editValue(setting), setting.currentValue, privacyValueType(setting))
  }

  function changeForSetting(setting: PrivacyConfigSetting): PrivacyConfigChange {
    const edit = editFor(setting)
    if (edit.op === 'unset') return { id: setting.id, op: 'unset' }
    return { id: setting.id, op: 'set', value: editValue(setting) }
  }

  function canEdit(setting: PrivacyConfigSetting) {
    return setting.canApply !== false
  }

  return {
    edits,
    syncEdits,
    editFor,
    editValue,
    markEditSet,
    useStrict,
    unsetEdit,
    resetEdit,
    isEditChanged,
    changeForSetting,
    canEdit
  }
}
