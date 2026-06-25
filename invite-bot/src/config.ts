export interface BotConfig {
    httpAddr: string
    internalSecret: string
    botToken: string
    guildId: string
    verifiedRoleId: string
    auditReason: string
}

function required(name: string): string {
    const value = process.env[name]
    if (value === undefined || value.trim() === '') {
        throw new Error(`missing required env: ${name}`)
    }

    return value.trim()
}

function optional(name: string, fallback: string): string {
    const value = process.env[name]
    if (value === undefined || value.trim() === '') {
        return fallback
    }

    return value.trim()
}

export function loadDotenv(path = '.env'): void {
    const loader = (process as unknown as { loadEnvFile?: (p: string) => void }).loadEnvFile
    if (typeof loader !== 'function') {
        return
    }

    try {
        loader(path)
    } catch {
        return
    }
}

export function loadConfig(): BotConfig {
    return {
        httpAddr: optional('HTTP_ADDR', ':8083'),
        internalSecret: required('DISCORD_INTERNAL_SECRET'),
        botToken: required('DISCORD_BOT_TOKEN'),
        guildId: required('DISCORD_GUILD_ID'),
        verifiedRoleId: required('DISCORD_VERIFIED_ROLE_ID'),
        auditReason: optional('DISCORD_AUDIT_REASON', 'External site verification'),
    }
}

export function parsePort(httpAddr: string): number {
    const idx = httpAddr.lastIndexOf(':')
    const raw = idx >= 0 ? httpAddr.slice(idx + 1) : httpAddr
    const port = Number.parseInt(raw, 10)
    if (Number.isNaN(port) || port <= 0) {
        throw new Error(`invalid HTTP_ADDR: ${httpAddr}`)
    }

    return port
}
