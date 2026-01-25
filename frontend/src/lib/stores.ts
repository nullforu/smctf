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
