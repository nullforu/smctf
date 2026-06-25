export type BotErrorCode = 'NOT_IN_GUILD' | 'BOT_PERMISSION' | 'RATE_LIMITED' | 'INVALID' | 'UPSTREAM'

export class BotError extends Error {
    readonly status: number
    readonly code: BotErrorCode

    constructor(status: number, code: BotErrorCode, message: string) {
        super(message)
        this.name = 'BotError'
        this.status = status
        this.code = code
    }
}

const UNKNOWN_MEMBER = 10007
const UNKNOWN_USER = 10013

interface DiscordLikeError {
    status?: number
    code?: number | string
    message?: string
}

export function mapDiscordError(err: unknown): BotError {
    const e = err as DiscordLikeError
    const message = e?.message ?? 'discord api error'

    if (e?.code === UNKNOWN_MEMBER || e?.code === UNKNOWN_USER) {
        return new BotError(404, 'NOT_IN_GUILD', message)
    }

    switch (e?.status) {
        case 404:
            return new BotError(404, 'NOT_IN_GUILD', message)
        case 403:
            return new BotError(403, 'BOT_PERMISSION', message)
        case 429:
            return new BotError(429, 'RATE_LIMITED', message)
        case 400:
            return new BotError(400, 'INVALID', message)
        default:
            return new BotError(502, 'UPSTREAM', message)
    }
}
