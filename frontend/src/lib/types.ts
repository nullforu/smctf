export interface AuthUser {
    id: number
    email: string
    username: string
    role: string
}

export interface RegisterPayload {
    email: string
    username: string
    password: string
    registration_key: string
}

export interface RegisterResponse {
    id: number
    email: string
    username: string
}

export interface LoginPayload {
    email: string
    password: string
}

export interface AuthResponse {
    access_token: string
    refresh_token: string
    user: AuthUser
}

export interface Challenge {
    id: number
    title: string
    description: string
    category: string
    points: number
    is_active: boolean
}

export interface ChallengeCreatePayload {
    title: string
    description: string
    category: string
    points: number
    flag: string
    is_active: boolean
}

export interface ChallengeCreateResponse extends Challenge {}

export interface ChallengeUpdatePayload {
    title?: string
    description?: string
    category?: string
    points?: number
    is_active?: boolean
}

export interface FlagSubmissionPayload {
    flag: string
}

export interface FlagSubmissionResult {
    correct: boolean
}

export interface SolvedChallenge {
    challenge_id: number
    title: string
    points: number
    solved_at: string
}

export interface ScoreEntry {
    user_id: number
    username: string
    score: number
}

export interface TimelineSubmission {
    timestamp: string
    user_id: number
    username: string
    points: number
    challenge_count: number
}

export interface TimelineResponse {
    submissions: TimelineSubmission[]
}

export interface UserListItem {
    id: number
    username: string
    role: string
}

export interface UserDetail {
    id: number
    username: string
    role: string
}

export interface RegistrationKey {
    id: number
    code: string
    created_by: number
    created_by_username: string
    used_by?: number
    used_by_username?: string
    used_by_ip?: string
    created_at: string
    used_at?: string
}

export interface RegistrationKeyCreatePayload {
    count: number
}
