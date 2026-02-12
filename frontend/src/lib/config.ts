import { get, writable } from 'svelte/store'
import { api } from './api'
import { localeStore, t } from './i18n'

export interface AppConfigState {
    title: string
    description: string
    header_title: string
    header_description: string
    updated_at?: string
}

const buildDefaultConfig = (): AppConfigState => {
    const translate = get(t)
    return {
        title: translate('config.defaultTitle'),
        description: translate('config.defaultDescription'),
        header_title: translate('config.defaultHeaderTitle'),
        header_description: translate('config.defaultHeaderDescription'),
    }
}

let defaultConfig: AppConfigState = buildDefaultConfig()

export const configStore = writable<AppConfigState>(defaultConfig)

let loaded = false
let inFlight: Promise<void> | null = null
let hasLoadedRemote = false

localeStore.subscribe(() => {
    defaultConfig = buildDefaultConfig()
    if (!hasLoadedRemote) {
        configStore.set(defaultConfig)
    }
})

export const setConfig = (config: AppConfigState) => {
    hasLoadedRemote = true
    configStore.set({
        title: config.title ?? defaultConfig.title,
        description: config.description ?? defaultConfig.description,
        header_title: config.header_title ?? defaultConfig.header_title,
        header_description: config.header_description ?? defaultConfig.header_description,
        updated_at: config.updated_at,
    })
}

export const loadConfig = async () => {
    if (loaded) return
    if (inFlight) return inFlight

    inFlight = (async () => {
        try {
            const config = await api.config()
            setConfig(config)
        } catch {
            hasLoadedRemote = false
            configStore.set(defaultConfig)
        } finally {
            loaded = true
            inFlight = null
        }
    })()

    return inFlight
}

export const resetConfig = () => {
    loaded = false
    inFlight = null
    hasLoadedRemote = false
    configStore.set(defaultConfig)
}
