import type { Server } from 'node:http'

import { loadConfig, loadDotenv, parsePort } from './config.js'
import { DiscordBot } from './discord/client.js'
import { buildServer } from './http/server.js'
import { logger } from './logger.js'

async function main(): Promise<void> {
    loadDotenv()
    const cfg = loadConfig()
    const port = parsePort(cfg.httpAddr)

    const bot = new DiscordBot(cfg)
    await bot.start()

    const app = buildServer(cfg, bot)
    const server: Server = app.listen(port, () => {
        logger.info('discord bot http server listening', { port })
    })

    const shutdown = (signal: string) => {
        logger.info('shutting down', { signal })
        server.close(() => {
            void bot.stop().finally(() => process.exit(0))
        })
    }

    process.on('SIGTERM', () => shutdown('SIGTERM'))
    process.on('SIGINT', () => shutdown('SIGINT'))
}

main().catch((err: unknown) => {
    logger.error('fatal startup error', {
        error: err instanceof Error ? err.message : String(err),
    })

    process.exit(1)
})
