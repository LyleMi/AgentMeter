import { computed, readonly, ref, watch } from 'vue'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import type { Locale as AntDesignLocale } from 'ant-design-vue/es/locale'

export type AppLocale = 'en' | 'zh-CN'

type MessageDictionary = Record<string, string>
type LocaleMessageMap = Record<AppLocale, MessageDictionary>
type InterpolationValue = string | number | boolean | null | undefined
type InterpolationParams = Record<string, InterpolationValue>

const localeStorageKey = 'agentmeter.locale'
const fallbackLocale: AppLocale = 'en'
const supportedLocales = ['en', 'zh-CN'] as const
const antLocales: Record<AppLocale, AntDesignLocale> = {
  en: enUS,
  'zh-CN': zhCN
}

export const localeOptions = supportedLocales.map((value) => ({ value }))

function readStoredLocale() {
  if (typeof window === 'undefined') return undefined

  try {
    return window.localStorage.getItem(localeStorageKey)
  } catch {
    return undefined
  }
}

function writeStoredLocale(nextLocale: AppLocale) {
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(localeStorageKey, nextLocale)
  } catch {
    // Ignore unavailable storage; the in-memory locale still updates.
  }
}

export function normalizeLocale(value: string | null | undefined): AppLocale | undefined {
  const normalized = value?.trim().replace('_', '-').toLowerCase()
  if (!normalized) return undefined
  if (normalized === 'zh-cn' || normalized === 'zh-hans' || normalized.startsWith('zh-')) return 'zh-CN'
  if (normalized === 'en' || normalized.startsWith('en-')) return 'en'
  return undefined
}

function detectLocale(): AppLocale {
  const storedLocale = normalizeLocale(readStoredLocale())
  if (storedLocale) return storedLocale

  if (typeof navigator !== 'undefined') {
    const languages = [...(navigator.languages || []), navigator.language]
    for (const language of languages) {
      const detectedLocale = normalizeLocale(language)
      if (detectedLocale) return detectedLocale
    }
  }

  return fallbackLocale
}

const activeLocale = ref<AppLocale>(detectLocale())

export const currentLocale = readonly(activeLocale)
export const intlLocale = computed(() => (activeLocale.value === 'zh-CN' ? 'zh-CN' : 'en-US'))
export const antDesignLocale = computed(() => antLocales[activeLocale.value])

export function setLocale(nextLocale: AppLocale | string) {
  const normalizedLocale = normalizeLocale(nextLocale)
  if (!normalizedLocale || normalizedLocale === activeLocale.value) return
  activeLocale.value = normalizedLocale
}

export function createNumberFormatter(options?: Intl.NumberFormatOptions) {
  return new Intl.NumberFormat(intlLocale.value, options)
}

export function createDateTimeFormatter(options?: Intl.DateTimeFormatOptions) {
  return new Intl.DateTimeFormat(intlLocale.value, options)
}

export function formatNumber(value: number | bigint, options?: Intl.NumberFormatOptions) {
  return createNumberFormatter(options).format(value)
}

export function formatDateTime(value: string | number | Date, options?: Intl.DateTimeFormatOptions) {
  const date = value instanceof Date ? value : new Date(value)
  return createDateTimeFormatter(options).format(date)
}

function interpolate(template: string, params?: InterpolationParams) {
  if (!params) return template

  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, key: string) => {
    const value = params[key]
    return value === null || value === undefined ? '' : String(value)
  })
}

export function useMessages<const Messages extends LocaleMessageMap>(messages: Messages) {
  type MessageKey = Extract<keyof Messages['en'] | keyof Messages['zh-CN'], string>

  function t(key: MessageKey, params?: InterpolationParams) {
    const localizedMessage = messages[activeLocale.value]?.[key] ?? messages[fallbackLocale]?.[key] ?? key
    return interpolate(localizedMessage, params)
  }

  return {
    t,
    locale: currentLocale,
    intlLocale,
    setLocale,
    formatNumber,
    formatDateTime,
    createNumberFormatter,
    createDateTimeFormatter
  }
}

watch(
  activeLocale,
  (nextLocale) => {
    writeStoredLocale(nextLocale)
    if (typeof document !== 'undefined') {
      document.documentElement.lang = nextLocale
    }
  },
  { immediate: true }
)
