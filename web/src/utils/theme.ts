export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

export interface ThemeSnapshot {
    preference: ThemePreference
    effective: ResolvedTheme
}

export const THEME_STORAGE_KEY = 'quickstart.theme.preference'
const SYSTEM_QUERY = '(prefers-color-scheme: dark)'
const listeners = new Set<(snapshot: ThemeSnapshot) => void>()

let preference: ThemePreference | null = null
let effective: ResolvedTheme = 'light'
let media: MediaQueryList | null = null
let observer: MutationObserver | null = null
let started = false

export function normalizeThemePreference(value: unknown): ThemePreference | null {
    if (value === 'system' || value === 'normal') return 'system'
    if (value === 'light' || value === 'dark') return value
    return null
}

export function resolveTheme(value: ThemePreference, systemDark: boolean): ResolvedTheme {
    return value === 'system' ? (systemDark ? 'dark' : 'light') : value
}

export function bodyThemeForPreference(value: ThemePreference): 'normal' | 'light' | 'dark' {
    return value === 'system' ? 'normal' : value
}

function storedPreference(): ThemePreference | null {
    try {
        return normalizeThemePreference(window.localStorage.getItem(THEME_STORAGE_KEY))
    } catch {
        return null
    }
}

function preferenceFromBody(): ThemePreference {
    return normalizeThemePreference(document.body.getAttribute('theme')) || 'system'
}

function persist(value: ThemePreference) {
    try {
        window.localStorage.setItem(THEME_STORAGE_KEY, value)
    } catch {
        // Theme selection remains usable when browser storage is unavailable.
    }
}

function notify() {
    const snapshot = { preference: preference!, effective }
    listeners.forEach(listener => listener(snapshot))
    window.dispatchEvent(new CustomEvent<ThemeSnapshot>('quickstart-theme-change', { detail: snapshot }))
}

function syncResolved(force = false) {
    const next = resolveTheme(preference!, media?.matches ?? window.matchMedia(SYSTEM_QUERY).matches)
    const changed = next !== effective
    effective = next
    document.body.dataset.quickstartTheme = effective
    document.documentElement.dataset.quickstartTheme = effective
    document.documentElement.style.colorScheme = effective
    if (changed || force) notify()
}

function applyBodyPreference() {
    const value = bodyThemeForPreference(preference!)
    if (document.body.getAttribute('theme') !== value) document.body.setAttribute('theme', value)
}

function mediaChanged() {
    if (preference === 'system') syncResolved()
}

function bodyChanged() {
    const next = preferenceFromBody()
    if (next === preference) return
    preference = next
    persist(next)
    syncResolved(true)
}

function storageChanged(event: StorageEvent) {
    if (event.key !== THEME_STORAGE_KEY) return
    const next = normalizeThemePreference(event.newValue) || 'system'
    if (next === preference) return
    preference = next
    applyBodyPreference()
    syncResolved(true)
}

export function initializeTheme(): ThemeSnapshot {
    if (started) return { preference: preference!, effective }
    started = true
    media = window.matchMedia(SYSTEM_QUERY)
    preference = storedPreference() || preferenceFromBody()
    applyBodyPreference()
    syncResolved(true)
    media.addEventListener('change', mediaChanged)
    observer = new MutationObserver(bodyChanged)
    observer.observe(document.body, { attributes: true, attributeFilter: ['theme'] })
    window.addEventListener('storage', storageChanged)
    return { preference, effective }
}

export function getThemePreference(): ThemePreference {
    return initializeTheme().preference
}

export function getTheme(): ResolvedTheme {
    return initializeTheme().effective
}

export function setThemePreference(value: ThemePreference): ThemeSnapshot {
    initializeTheme()
    preference = value
    persist(value)
    applyBodyPreference()
    syncResolved(true)
    return { preference, effective }
}

export function subscribeTheme(listener: (snapshot: ThemeSnapshot) => void): () => void {
    const snapshot = initializeTheme()
    listeners.add(listener)
    listener(snapshot)
    return () => listeners.delete(listener)
}

export const isDark = () => getTheme() === 'dark'
export const isLight = () => getTheme() === 'light'
