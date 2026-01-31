export interface AuthUser {
    id: number
    email: string
    username: string
    role: string
    team_id?: number | null
    team_name?: string | null
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
    initial_points: number
    minimum_points: number
    solve_count: number
    is_active: boolean
}

export interface ChallengeCreatePayload {
    title: string
    description: string
    category: string
    points: number
    minimum_points?: number
    flag: string
    is_active: boolean
}

export interface ChallengeCreateResponse extends Challenge {}

export interface ChallengeUpdatePayload {
    title?: string
    description?: string
    category?: string
    points?: number
    minimum_points?: number
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

export interface TeamScoreEntry {
    team_id?: number | null
    team_name: string
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

export interface TeamTimelineSubmission {
    timestamp: string
    team_id?: number | null
    team_name: string
    points: number
    challenge_count: number
}

export interface TeamTimelineResponse {
    submissions: TeamTimelineSubmission[]
}

export interface UserListItem {
    id: number
    username: string
    role: string
    team_id?: number | null
    team_name?: string | null
}

export interface UserDetail {
    id: number
    username: string
    role: string
    team_id?: number | null
    team_name?: string | null
}

export interface RegistrationKey {
    id: number
    code: string
    created_by: number
    created_by_username: string
    team_id?: number | null
    team_name?: string | null
    used_by?: number
    used_by_username?: string
    used_by_ip?: string
    created_at: string
    used_at?: string
}

export interface RegistrationKeyCreatePayload {
    count: number
    team_id?: number | null
}

export interface Team {
    id: number
    name: string
    created_at: string
}

export interface TeamCreatePayload {
    name: string
}

export interface TeamSummary {
    id: number
    name: string
    created_at: string
    member_count: number
    total_score: number
}

export interface TeamMember {
    id: number
    username: string
    role: string
}

export interface TeamDetail extends TeamSummary {}

export interface TeamSolvedChallenge {
    challenge_id: number
    title: string
    points: number
    solve_count: number
    last_solved_at: string
}
