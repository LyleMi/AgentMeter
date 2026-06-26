export type AppDataChangeReason = 'settings' | 'index'

export interface AppDataChangeDetail {
  reason: AppDataChangeReason
}

export const APP_DATA_CHANGED_EVENT = 'agentmeter:data-changed'

export function notifyAppDataChanged(reason: AppDataChangeReason) {
  window.dispatchEvent(new CustomEvent<AppDataChangeDetail>(APP_DATA_CHANGED_EVENT, { detail: { reason } }))
}
