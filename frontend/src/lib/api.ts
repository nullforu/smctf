import { get } from 'svelte/store'
import { authStore, clearAuth, setAuthTokens, setAuthUser } from './stores'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

export type ApiErrorDetail = { field: string; reason: string }
export type RateLimitInfo = { limit: number; remaining: number; reset_seconds: number }

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
    if (!contentType.includes('application/json')) {
        return null
    }
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
    const headers: Record<string, string> = {
        Accept: 'application/json',
    }
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
    if (body !== undefined) {
        headers['Content-Type'] = 'application/json'
    }

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
                const data = await parseJson(retryResponse)
                return data as T
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

export type AuthResponse = {
    access_token: string
    refresh_token: string
    user: { id: number; email: string; username: string; role: string }
}

export const api = {
    register: (payload: { email: string; username: string; password: string }) =>
        request<{ id: number; email: string; username: string }>(`/api/auth/register`, {
            method: 'POST',
            body: payload,
        }),
    login: async (payload: { email: string; password: string }) => {
        const data = await request<AuthResponse>(`/api/auth/login`, {
            method: 'POST',
            body: payload,
        })
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
        await request(`/api/auth/logout`, {
            method: 'POST',
            body: { refresh_token: refreshTokenValue },
        })
        clearAuth()
    },
    me: () => request<{ id: number; email: string; username: string; role: string }>(`/api/me`, { auth: true }),
    solved: () =>
        request<Array<{ challenge_id: number; title: string; points: number; solved_at: string }>>(`/api/me/solved`, {
            auth: true,
        }),
    challenges: () =>
        request<Array<{ id: number; title: string; description: string; points: number; is_active: boolean }>>(
            `/api/challenges`,
        ),
    submitFlag: (id: number, flag: string) =>
        request<{ correct: boolean }>(`/api/challenges/${id}/submit`, {
            method: 'POST',
            body: { flag },
            auth: true,
        }),
    scoreboard: (limit = 50) =>
        request<Array<{ user_id: number; username: string; score: number }>>(`/api/scoreboard?limit=${limit}`),
    timeline: (interval = 10, limit = 50) =>
        request<{
            interval_minutes: number
            users: Array<{ user_id: number; username: string; score: number }>
            buckets: Array<{
                bucket: string
                scores: Array<{ user_id: number; username: string; score: number }>
            }>
        }>(`/api/scoreboard/timeline?interval=${interval}&limit=${limit}`),
    createChallenge: (payload: {
        title: string
        description: string
        points: number
        flag: string
        is_active: boolean
    }) =>
        request<{ id: number; title: string; description: string; points: number; is_active: boolean }>(
            `/api/admin/challenges`,
            { method: 'POST', body: payload, auth: true },
        ),
}
