import { writable } from 'svelte/store'

export type User = {
    id: number
    email: string
    username: string
    role: string
}

export type AuthState = {
    accessToken: string | null
    refreshToken: string | null
    user: User | null
}

const STORAGE_KEY = 'smctf.auth'

const loadAuth = (): AuthState => {
    if (typeof localStorage === 'undefined') {
        return { accessToken: null, refreshToken: null, user: null }
    }
    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (!raw) return { accessToken: null, refreshToken: null, user: null }
        const parsed = JSON.parse(raw) as AuthState
        return {
            accessToken: parsed.accessToken ?? null,
            refreshToken: parsed.refreshToken ?? null,
            user: parsed.user ?? null,
        }
    } catch {
        return { accessToken: null, refreshToken: null, user: null }
    }
}

const persistAuth = (state: AuthState) => {
    if (typeof localStorage === 'undefined') return
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

export const authStore = writable<AuthState>(loadAuth())

authStore.subscribe((value) => {
    persistAuth(value)
})

export const setAuthTokens = (accessToken: string, refreshToken: string) => {
    authStore.update((state) => ({ ...state, accessToken, refreshToken }))
}

export const setAuthUser = (user: User | null) => {
    authStore.update((state) => ({ ...state, user }))
}

export const clearAuth = () => {
    authStore.set({ accessToken: null, refreshToken: null, user: null })
}
