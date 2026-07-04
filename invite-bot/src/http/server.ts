import express, { type Express, type NextFunction, type Request, type Response } from 'express'

import type { BotConfig } from '../config.js'
import type { DiscordBot } from '../discord/client.js'
import { BotError } from '../discord/errors.js'
import { logger } from '../logger.js'

function secretsMatch(a: string, b: string): boolean {
    if (a.length !== b.length) {
        return false
    }

    let diff = 0
    for (let i = 0; i < a.length; i += 1) {
        diff |= a.charCodeAt(i) ^ b.charCodeAt(i)
    }

    return diff === 0
}

function requireSecret(secret: string) {
    return (req: Request, res: Response, next: NextFunction): void => {
        const header = req.header('authorization') ?? ''
        const expected = `Bearer ${secret}`
        if (!secretsMatch(header, expected)) {
            res.status(401).json({ error: 'unauthorized', code: 'INVALID' })
            return
        }
        next()
    }
}

function pathUserId(req: Request): string {
    const raw = req.params.userId
    const value = Array.isArray(raw) ? raw[0] : raw
    if (typeof value !== 'string' || !/^\d{1,32}$/.test(value)) {
        throw new BotError(400, 'INVALID', 'invalid user id')
    }

    return value
}

function asyncRoute(fn: (req: Request, res: Response) => Promise<void>): (req: Request, res: Response) => void {
    return (req, res) => {
        fn(req, res).catch((err: unknown) => {
            if (err instanceof BotError) {
                res.status(err.status).json({ error: err.message, code: err.code })
                return
            }

            logger.error('unhandled route error', {
                path: req.path,
                error: err instanceof Error ? err.message : String(err),
            })

            res.status(500).json({ error: 'internal error', code: 'UPSTREAM' })
        })
    }
}

export function buildServer(cfg: BotConfig, bot: DiscordBot): Express {
    const app = express()
    app.use(express.json())

    app.get('/healthz', (_req, res) => {
        res.json({ status: 'ok' })
    })

    const internal = express.Router()
    internal.use(requireSecret(cfg.internalSecret))

    internal.put(
        '/guild/members/:userId',
        asyncRoute(async (req, res) => {
            const accessToken = (req.body?.access_token as string | undefined) ?? ''
            if (accessToken.trim() === '') {
                res.status(400).json({ error: 'access_token required', code: 'INVALID' })
                return
            }

            const roleId = (req.body?.role_id as string | undefined)?.trim() ?? ''
            await bot.joinGuild(pathUserId(req), accessToken, roleId)
            res.status(204).end()
        }),
    )

    internal.put(
        '/guild/members/:userId/role',
        asyncRoute(async (req, res) => {
            const roleId = (req.body?.role_id as string | undefined)?.trim() ?? ''
            await bot.grantRole(pathUserId(req), roleId)
            res.status(204).end()
        }),
    )

    internal.delete(
        '/guild/members/:userId',
        asyncRoute(async (req, res) => {
            await bot.kickMember(pathUserId(req))
            res.status(204).end()
        }),
    )

    internal.get(
        '/guild/members/:userId',
        asyncRoute(async (req, res) => {
            const rawRole = req.query.role_id
            const roleId = typeof rawRole === 'string' ? rawRole.trim() : ''
            const status = await bot.memberStatus(pathUserId(req), roleId)
            res.json(status)
        }),
    )

    internal.post(
        '/announce',
        asyncRoute(async (req, res) => {
            const content = (req.body?.content as string | undefined) ?? ''
            if (content.trim() === '') {
                res.status(400).json({ error: 'content required', code: 'INVALID' })
                return
            }
            const channelId = (req.body?.channel_id as string | undefined)?.trim() ?? ''
            await bot.announce(channelId, content)
            res.status(204).end()
        }),
    )

    internal.patch(
        '/guild/members/:userId/nickname',
        asyncRoute(async (req, res) => {
            const nickname = (req.body?.nickname as string | undefined) ?? ''
            await bot.setNickname(pathUserId(req), nickname)
            res.status(204).end()
        }),
    )

    app.use('/internal', internal)

    return app
}
