import { writable } from 'svelte/store'
import { api } from './api'

export interface AppConfigState {
    title: string
    description: string
    header_title: string
    header_description: string
    updated_at?: string
}

const defaultConfig: AppConfigState = {
    title: 'Welcome to SMCTF.',
    description: 'Check out the repository for setup instructions.',
    header_title: 'CTF',
    header_description: 'Capture The Flag',
}

export const configStore = writable<AppConfigState>(defaultConfig)

let loaded = false
let inFlight: Promise<void> | null = null

export const setConfig = (config: AppConfigState) => {
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
            setConfig(defaultConfig)
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
    setConfig(defaultConfig)
}
