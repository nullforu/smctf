import { Client, GatewayIntentBits, Guild, GuildMember, DiscordAPIError } from 'discord.js'

import type { BotConfig } from '../config.js'
import { logger } from '../logger.js'
import { BotError, mapDiscordError } from './errors.js'

const UNKNOWN_MEMBER = 10007

export interface MemberStatus {
    in_guild: boolean
    has_role: boolean
}

export class DiscordBot {
    private readonly client: Client
    private guild: Guild | null = null

    constructor(private readonly cfg: BotConfig) {
        this.client = new Client({
            intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildMembers],
        })
    }

    async start(): Promise<void> {
        await this.client.login(this.cfg.botToken)
        this.guild = await this.client.guilds.fetch(this.cfg.guildId)

        this.client.on('guildMemberRemove', (member) => {
            logger.info('guild member left', { discord_user_id: member.id })
        })

        logger.info('discord bot ready', { guild_id: this.cfg.guildId })
    }

    async stop(): Promise<void> {
        await this.client.destroy()
    }

    private requireGuild(): Guild {
        if (!this.guild) {
            throw new BotError(503, 'UPSTREAM', 'guild not ready')
        }
        return this.guild
    }

    async joinGuild(userId: string, accessToken: string, roleId: string): Promise<void> {
        const guild = this.requireGuild()
        try {
            await guild.members.add(userId, {
                accessToken,
                roles: roleId ? [roleId] : [],
            })
        } catch (err) {
            throw mapDiscordError(err)
        }
    }

    async grantRole(userId: string, roleId: string): Promise<void> {
        if (!roleId) {
            return
        }
        const member = await this.fetchMember(userId)
        try {
            await member.roles.add(roleId, this.cfg.auditReason)
        } catch (err) {
            throw mapDiscordError(err)
        }
    }

    async kickMember(userId: string): Promise<void> {
        const guild = this.requireGuild()
        try {
            await guild.members.kick(userId, this.cfg.auditReason)
        } catch (err) {
            if (err instanceof DiscordAPIError && err.code === UNKNOWN_MEMBER) {
                return
            }
            throw mapDiscordError(err)
        }
    }

    async memberStatus(userId: string, roleId: string): Promise<MemberStatus> {
        try {
            const member = await this.fetchMember(userId)
            return {
                in_guild: true,
                has_role: roleId ? member.roles.cache.has(roleId) : false,
            }
        } catch (err) {
            if (err instanceof BotError && err.code === 'NOT_IN_GUILD') {
                return { in_guild: false, has_role: false }
            }
            throw err
        }
    }

    async announce(channelId: string, content: string): Promise<void> {
        if (!channelId) {
            return
        }

        try {
            const channel = await this.client.channels.fetch(channelId)
            if (!channel || !channel.isTextBased() || !('send' in channel)) {
                throw new BotError(400, 'INVALID', 'announce channel is not text-based')
            }
            await channel.send({ content, allowedMentions: { parse: [] } })
        } catch (err) {
            if (err instanceof BotError) {
                throw err
            }
            throw mapDiscordError(err)
        }
    }

    async setNickname(userId: string, nickname: string): Promise<void> {
        const member = await this.fetchMember(userId)
        try {
            await member.setNickname(nickname.slice(0, 32), this.cfg.auditReason)
        } catch (err) {
            throw mapDiscordError(err)
        }
    }

    private async fetchMember(userId: string): Promise<GuildMember> {
        const guild = this.requireGuild()
        try {
            return await guild.members.fetch(userId)
        } catch (err) {
            if (err instanceof DiscordAPIError && err.code === UNKNOWN_MEMBER) {
                throw new BotError(404, 'NOT_IN_GUILD', 'user not in guild')
            }
            throw mapDiscordError(err)
        }
    }
}
