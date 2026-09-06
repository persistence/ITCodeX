export type ThemeMode = 'light' | 'dark' | 'system'
export type TableDensity = 'default' | 'compact'

export interface AppSettings {
  apiBaseUrl: string
  theme: ThemeMode
  defaultPageSize: number
  tableDensity: TableDensity
}

const STORAGE_KEY = 'itcodex.meta.settings'

const defaults: AppSettings = {
  apiBaseUrl: '',
  theme: 'light',
  defaultPageSize: 20,
  tableDensity: 'default',
}

export function getSettings(): AppSettings {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaults }
    return { ...defaults, ...JSON.parse(raw) }
  } catch {
    return { ...defaults }
  }
}

export function saveSettings(partial: Partial<AppSettings>): AppSettings {
  const next = { ...getSettings(), ...partial }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  window.dispatchEvent(new CustomEvent('itcodex:settings', { detail: next }))
  return next
}

export function resolveTheme(theme: ThemeMode): 'light' | 'dark' {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }
  return theme
}
