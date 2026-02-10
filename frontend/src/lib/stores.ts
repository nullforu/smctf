import { writable } from 'svelte/store'
import type { AuthUser } from './types'

export interface AuthState {
    accessToken: string | null
    refreshToken: string | null
    user: AuthUser | null
}

const STORAGE_KEY = 'smctf.auth'

const emptyAuth = (): AuthState => ({ accessToken: null, refreshToken: null, user: null })

const loadAuth = (): AuthState => {
    if (typeof localStorage === 'undefined') return emptyAuth()

    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (!raw) return emptyAuth()

        return JSON.parse(raw) as AuthState
    } catch {
        return emptyAuth()
    }
}

const persistAuth = (state: AuthState) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
    }
}

export const authStore = writable<AuthState>(loadAuth())

authStore.subscribe(persistAuth)

export const setAuthTokens = (accessToken: string, refreshToken: string) => {
    authStore.update((state) => ({ ...state, accessToken, refreshToken }))
}

export const setAuthUser = (user: AuthUser | null) => {
    authStore.update((state) => ({ ...state, user }))
}

export const clearAuth = () => {
    authStore.set(emptyAuth())
}

const THEME_KEY = 'smctf.theme'

type Theme = 'light' | 'dark'

const loadTheme = (): Theme => {
    if (typeof localStorage === 'undefined') return 'light'

    try {
        const saved = localStorage.getItem(THEME_KEY)
        return saved === 'dark' ? 'dark' : 'light'
    } catch {
        return 'light'
    }
}

const persistTheme = (theme: Theme) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(THEME_KEY, theme)
    }
}

export const themeStore = writable<Theme>(loadTheme())

themeStore.subscribe(persistTheme)

export const toggleThemeValue = (theme: Theme): Theme => (theme === 'light' ? 'dark' : 'light')

export const toggleTheme = () => {
    themeStore.update(toggleThemeValue)
}

