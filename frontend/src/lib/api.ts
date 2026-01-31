import { get } from 'svelte/store'
import { authStore, clearAuth, setAuthTokens, setAuthUser } from './stores'
import type {
    AuthResponse,
    AuthUser,
    AppConfig,
    AdminConfigUpdatePayload,
    Challenge,
    ChallengeCreatePayload,
    ChallengeCreateResponse,
    ChallengeUpdatePayload,
    FlagSubmissionResult,
    Team,
    TeamCreatePayload,
    TeamScoreEntry,
    TeamSummary,
    TeamDetail,
    TeamMember,
    TeamSolvedChallenge,
    TeamTimelineResponse,
    LoginPayload,
    RegistrationKey,
    RegistrationKeyCreatePayload,
    RegisterPayload,
    RegisterResponse,
    ScoreEntry,
    SolvedChallenge,
    TimelineResponse,
    UserListItem,
    UserDetail,
} from './types'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

export interface ApiErrorDetail {
    field: string
    reason: string
}
export interface RateLimitInfo {
    limit: number
    remaining: number
    reset_seconds: number
}

export class ApiError extends Error {
    status: number
    details?: ApiErrorDetail[]
    rateLimit?: RateLimitInfo

    constructor(message: string, status: number, details?: ApiErrorDetail[], rateLimit?: RateLimitInfo) {
        super(message)
        this.name = 'ApiError'
        this.status = status
        this.details = details
        this.rateLimit = rateLimit
    }
}

const parseJson = async (response: Response) => {
    const contentType = response.headers.get('content-type') ?? ''
    if (!contentType.includes('application/json')) return null

    return response.json()
}

const extractRateLimit = (response: Response, data: any): RateLimitInfo | undefined => {
    if (data?.rate_limit) return data.rate_limit as RateLimitInfo

    const limit = Number(response.headers.get('x-ratelimit-limit'))
    const remaining = Number(response.headers.get('x-ratelimit-remaining'))
    const resetSeconds = Number(response.headers.get('x-ratelimit-reset'))

    if (Number.isFinite(limit) && Number.isFinite(remaining) && Number.isFinite(resetSeconds)) {
        return { limit, remaining, reset_seconds: resetSeconds }
    }

    return undefined
}

const buildHeaders = (withAuth: boolean, tokenOverride?: string) => {
    const headers: Record<string, string> = { Accept: 'application/json' }

    if (withAuth) {
        const token = tokenOverride ?? get(authStore).accessToken
        if (token) headers.Authorization = `Bearer ${token}`
    }

    return headers
}

const refreshToken = async () => {
    const refreshTokenValue = get(authStore).refreshToken
    if (!refreshTokenValue) throw new ApiError('missing refresh token', 401)

    const response = await fetch(`${API_BASE}/api/auth/refresh`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            Accept: 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshTokenValue }),
    })

    if (!response.ok) {
        const data = await parseJson(response)
        clearAuth()

        throw new ApiError(
            data?.error ?? 'invalid credentials',
            response.status,
            data?.details,
            extractRateLimit(response, data),
        )
    }

    const data = await response.json()
    setAuthTokens(data.access_token, data.refresh_token)

    return data.access_token as string
}

const request = async <T>(
    path: string,
    {
        method = 'GET',
        body,
        auth = false,
        retryOnAuth = true,
    }: {
        method?: string
        body?: unknown
        auth?: boolean
        retryOnAuth?: boolean
    } = {},
): Promise<T> => {
    const headers = buildHeaders(auth)
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    const response = await fetch(`${API_BASE}${path}`, {
        method,
        headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
    })

    if (response.ok) {
        if (response.status === 204) return null as T
        const data = await parseJson(response)
        return data as T
    }

    if (response.status === 401 && auth && retryOnAuth) {
        try {
            const newToken = await refreshToken()
            const retryHeaders = buildHeaders(true, newToken)
            if (body !== undefined) retryHeaders['Content-Type'] = 'application/json'

            const retryResponse = await fetch(`${API_BASE}${path}`, {
                method,
                headers: retryHeaders,
                body: body !== undefined ? JSON.stringify(body) : undefined,
            })

            if (retryResponse.ok) {
                if (retryResponse.status === 204) return null as T
                return (await parseJson(retryResponse)) as T
            }

            const retryData = await parseJson(retryResponse)
            throw new ApiError(
                retryData?.error ?? 'request failed',
                retryResponse.status,
                retryData?.details,
                extractRateLimit(retryResponse, retryData),
            )
        } catch (error) {
            if (error instanceof ApiError) throw error
            clearAuth()
            throw new ApiError('invalid credentials', 401)
        }
    }

    const data = await parseJson(response)

    throw new ApiError(
        data?.error ?? 'request failed',
        response.status,
        data?.details,
        extractRateLimit(response, data),
    )
}

export const api = {
    config: () => {
        return request<AppConfig>(`/api/config`)
    },
    updateAdminConfig: (payload: AdminConfigUpdatePayload) => {
        return request<AppConfig>(`/api/admin/config`, { method: 'PUT', body: payload, auth: true })
    },
    register: (payload: RegisterPayload) => {
        return request<RegisterResponse>(`/api/auth/register`, { method: 'POST', body: payload })
    },
    login: async (payload: LoginPayload) => {
        const data = await request<AuthResponse>(`/api/auth/login`, { method: 'POST', body: payload })
        setAuthTokens(data.access_token, data.refresh_token)
        setAuthUser(data.user)
        return data
    },
    logout: async () => {
        const refreshTokenValue = get(authStore).refreshToken
        if (!refreshTokenValue) {
            clearAuth()
            return
        }
        await request(`/api/auth/logout`, { method: 'POST', body: { refresh_token: refreshTokenValue } })
        clearAuth()
    },
    me: () => {
        return request<AuthUser>(`/api/me`, { auth: true })
    },
    updateMe: (username: string) => {
        return request<AuthUser>(`/api/me`, { method: 'PUT', body: { username }, auth: true })
    },
    challenges: () => {
        return request<Challenge[]>(`/api/challenges`)
    },
    submitFlag: (id: number, flag: string) => {
        return request<FlagSubmissionResult>(`/api/challenges/${id}/submit`, {
            method: 'POST',
            body: { flag },
            auth: true,
        })
    },
    leaderboard: () => {
        return request<ScoreEntry[]>(`/api/leaderboard`)
    },
    leaderboardTeams: () => {
        return request<TeamScoreEntry[]>(`/api/leaderboard/teams`)
    },
    timeline: (windowMinutes?: number) => {
        const windowParam = typeof windowMinutes === 'number' ? `?window=${windowMinutes}` : ''
        return request<TimelineResponse>(`/api/timeline${windowParam}`)
    },
    timelineTeams: (windowMinutes?: number) => {
        const windowParam = typeof windowMinutes === 'number' ? `?window=${windowMinutes}` : ''
        return request<TeamTimelineResponse>(`/api/timeline/teams${windowParam}`)
    },
    createChallenge: (payload: ChallengeCreatePayload) => {
        return request<ChallengeCreateResponse>(`/api/admin/challenges`, { method: 'POST', body: payload, auth: true })
    },
    updateChallenge: (id: number, payload: ChallengeUpdatePayload) => {
        return request<Challenge>(`/api/admin/challenges/${id}`, { method: 'PUT', body: payload, auth: true })
    },
    deleteChallenge: (id: number) => {
        return request<void>(`/api/admin/challenges/${id}`, { method: 'DELETE', auth: true })
    },
    registrationKeys: () => {
        return request<RegistrationKey[]>(`/api/admin/registration-keys`, { auth: true })
    },
    createRegistrationKeys: (payload: RegistrationKeyCreatePayload) => {
        return request<RegistrationKey[]>(`/api/admin/registration-keys`, { method: 'POST', body: payload, auth: true })
    },
    createTeam: (payload: TeamCreatePayload) => {
        return request<Team>(`/api/admin/teams`, { method: 'POST', body: payload, auth: true })
    },
    teams: () => {
        return request<TeamSummary[]>(`/api/teams`)
    },
    teamDetail: (id: number) => {
        return request<TeamDetail>(`/api/teams/${id}`)
    },
    teamMembers: (id: number) => {
        return request<TeamMember[]>(`/api/teams/${id}/members`)
    },
    teamSolved: (id: number) => {
        return request<TeamSolvedChallenge[]>(`/api/teams/${id}/solved`)
    },
    users: () => {
        return request<UserListItem[]>(`/api/users`)
    },
    user: (id: number) => {
        return request<UserDetail>(`/api/users/${id}`)
    },
    userSolved: (id: number) => {
        return request<SolvedChallenge[]>(`/api/users/${id}/solved`)
    },
}
