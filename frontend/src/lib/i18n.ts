import { derived, get, writable } from 'svelte/store'
import en from '../locales/en.json'
import ko from '../locales/ko.json'
import jp from '../locales/jp.json'

export type Locale = 'en' | 'ko' | 'jp'

const STORAGE_KEY = 'smctf.locale'

const dictionaries: Record<Locale, Record<string, string>> = {
    en,
    ko,
    jp,
}

const normalizeLocale = (value?: string | null): Locale => {
    switch (value) {
        case 'ko':
            return 'ko'
        case 'jp':
            return 'jp'
        case 'en':
        default:
            return 'en'
    }
}

const loadLocale = (): Locale => {
    if (typeof localStorage === 'undefined') return 'en'
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) return normalizeLocale(saved)

    return 'en'
}

const persistLocale = (locale: Locale) => {
    if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, locale)
    }
}

export const localeStore = writable<Locale>(loadLocale())
localeStore.subscribe(persistLocale)

const interpolate = (message: string, vars?: Record<string, string | number>) => {
    if (!vars) return message
    return message.replace(/\{(\w+)\}/g, (_, key: string) => {
        const value = vars[key]
        return value === undefined || value === null ? '' : String(value)
    })
}

export const t = derived(localeStore, ($locale) => {
    return (key: string, vars?: Record<string, string | number>) => {
        const dictionary = dictionaries[$locale] ?? dictionaries.en
        const fallback = dictionaries.en
        const message = dictionary[key] ?? fallback[key] ?? key
        return interpolate(message, vars)
    }
})

export const setLocale = (locale: Locale) => {
    localeStore.set(locale)
}

export const getLocale = () => get(localeStore)

export const getLocaleTag = (locale: Locale) => {
    switch (locale) {
        case 'ko':
            return 'ko-KR'
        case 'jp':
            return 'ja-JP'
        case 'en':
        default:
            return 'en-US'
    }
}

const categoryKeyMap: Record<string, string> = {
    Web: 'categories.web',
    Web3: 'categories.web3',
    Pwnable: 'categories.pwnable',
    Reversing: 'categories.reversing',
    Crypto: 'categories.crypto',
    Forensics: 'categories.forensics',
    Network: 'categories.network',
    Cloud: 'categories.cloud',
    Misc: 'categories.misc',
    Programming: 'categories.programming',
    Algorithms: 'categories.algorithms',
    Math: 'categories.math',
    AI: 'categories.ai',
    Blockchain: 'categories.blockchain',
}

export const getCategoryKey = (category: string) => categoryKeyMap[category] ?? category

export const translateCategory = (category: string) => {
    const translate = get(t)
    return translate(getCategoryKey(category))
}

const roleKeyMap: Record<string, string> = {
    admin: 'roles.admin',
    user: 'roles.user',
}

export const getRoleKey = (role: string) => roleKeyMap[role] ?? role

export const translateRole = (role: string) => {
    const translate = get(t)
    return translate(getRoleKey(role))
}
