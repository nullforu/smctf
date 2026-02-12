<script lang="ts">
    import { untrack } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import type {
        LeaderboardChallenge,
        LeaderboardResponse,
        LeaderboardSolve,
        ScoreEntry,
        TeamLeaderboardResponse,
        TeamScoreEntry,
    } from '../lib/types'
    import { navigate } from '../lib/router'

    interface Props {
        mode?: 'users' | 'teams'
    }

    let { mode = 'users' }: Props = $props()

    type UserEntryView = ScoreEntry & { solveMap: Map<number, LeaderboardSolve> }
    type TeamEntryView = TeamScoreEntry & { solveMap: Map<number, LeaderboardSolve> }
    type EntryView = UserEntryView | TeamEntryView

    let challenges: LeaderboardChallenge[] = $state([])
    let scores: UserEntryView[] = $state([])
    let teamScores: TeamEntryView[] = $state([])
    let loading = $state(true)
    let errorMessage = $state('')
    let requestId = $state(0)
    const flagSize = 22
    const fixedCols = '48px 80px minmax(160px, 1fr)'

    const buildSolveMap = (solves: LeaderboardSolve[]) => {
        const map = new Map<number, LeaderboardSolve>()
        for (const solve of solves) {
            map.set(solve.challenge_id, solve)
        }
        return map
    }

    const applyUserLeaderboard = (payload: LeaderboardResponse) => {
        challenges = payload.challenges
        scores = payload.entries.map((entry) => ({
            ...entry,
            solveMap: buildSolveMap(entry.solves ?? []),
        }))
    }

    const applyTeamLeaderboard = (payload: TeamLeaderboardResponse) => {
        challenges = payload.challenges
        teamScores = payload.entries.map((entry) => ({
            ...entry,
            solveMap: buildSolveMap(entry.solves ?? []),
        }))
    }

    const entryLabel = (entry: EntryView) => {
        if ('team_name' in entry) {
            return entry.team_name
        }
        return entry.username
    }

    const entryHref = (entry: EntryView) => {
        if ('team_id' in entry) {
            return `/teams/${entry.team_id}`
        }
        return `/users/${entry.user_id}`
    }

    const gridTemplate = (count: number) => `${fixedCols} repeat(${count}, ${flagSize}px)`

    const loadScoreboard = async (modeValue: 'users' | 'teams') => {
        requestId += 1
        const currentRequest = requestId
        loading = true
        errorMessage = ''

        try {
            if (modeValue === 'teams') {
                const payload = await api.leaderboardTeams()
                if (currentRequest !== requestId) return
                applyTeamLeaderboard(payload)
            } else {
                const payload = await api.leaderboard()
                if (currentRequest !== requestId) return
                applyUserLeaderboard(payload)
            }
        } catch (error) {
            if (currentRequest === requestId) {
                errorMessage = formatApiError(error).message
            }
        } finally {
            if (currentRequest === requestId) {
                loading = false
            }
        }
    }

    $effect(() => {
        const selectedMode = mode
        untrack(() => {
            loadScoreboard(selectedMode)
        })
    })
</script>

<div
    class="min-w-0 rounded-2xl border border-slate-200 bg-white p-4 sm:p-6 dark:border-slate-800/80 dark:bg-slate-950/60"
>
    <div class="flex items-center justify-between">
        <h3 class="text-lg text-slate-900 dark:text-slate-100">
            {mode === 'teams' ? 'Team Leaderboard' : 'Leaderboard'}
        </h3>
        <span class="text-xs text-slate-500 dark:text-slate-400">{challenges.length} challenges</span>
    </div>
    {#if loading}
        <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">Loading...</p>
    {:else if errorMessage}
        <p class="mt-4 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
    {:else}
        <div class="mt-4 overflow-x-auto">
            <div class="min-w-max">
                <div
                    class="grid items-end gap-3 border-b border-slate-200 pb-3 text-[11px] uppercase tracking-wide text-slate-500 dark:border-slate-800/80 dark:text-slate-400"
                    style={`grid-template-columns: ${gridTemplate(challenges.length)};`}
                >
                    <span class="px-1">#</span>
                    <span class="px-1">Pts</span>
                    <span class="px-1">{mode === 'teams' ? 'Team' : 'User'}</span>
                    {#each challenges as challenge}
                        <span
                            class="relative inline-block h-[72px] w-[22px] text-[10px]"
                            title={`${challenge.title} (${challenge.points} pts)`}
                        >
                            <span
                                class="absolute bottom-0 left-0 block max-w-[15ch] overflow-hidden text-ellipsis whitespace-nowrap -rotate-[35deg] origin-bottom-left leading-none"
                            >
                                {challenge.title}
                            </span>
                        </span>
                    {/each}
                </div>

                <div class="divide-y divide-slate-200/70 dark:divide-slate-800/70">
                    {#each mode === 'teams' ? teamScores : scores as entry, index}
                        <button
                            class="grid w-full items-center gap-3 px-3 py-3 text-left transition hover:bg-slate-50 dark:hover:bg-slate-900/40"
                            style={`grid-template-columns: ${gridTemplate(challenges.length)};`}
                            onclick={() => navigate(entryHref(entry))}
                        >
                            <span class="text-xs text-slate-500">#{index + 1}</span>
                            <span class="text-xs font-semibold text-slate-700 dark:text-slate-300"
                                >{entry.score} pts</span
                            >
                            <span class="truncate text-sm text-slate-900 dark:text-slate-100">{entryLabel(entry)}</span>
                            {#each challenges as challenge}
                                {@const solve = entry.solveMap.get(challenge.id)}
                                <span
                                    class={`inline-flex h-[16px] w-[20px] items-center justify-center ${
                                        solve?.is_first_blood
                                            ? 'text-rose-500 dark:text-rose-400'
                                            : solve
                                              ? 'text-sky-500 dark:text-sky-400'
                                              : 'text-slate-300 dark:text-slate-600'
                                    }`}
                                    title={`${challenge.title}${solve ? (solve.is_first_blood ? ' • First Blood' : ' • Solved') : ' • Unsolved'}`}
                                >
                                    <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                                        <path
                                            d="M5 6.7c.9-.8 2.1-1.2 3.5-1.2 2.7 0 4.6 2.2 8.5.6v8.8c-3.9 1.7-5.8-.9-8.5-.9-1.2 0-2.5.3-3.5.9V6.7Z"
                                            fill="currentColor"
                                            opacity={solve ? '1' : '0.2'}
                                        />
                                        <path
                                            d="M4.5 21V16M4.5 16V6.5C5.5 5.5 7 5 8.5 5C11.5 5 13.5 7.5 17.5 5.5V15.5C13.5 17.5 11.5 14.5 8.5 14.5C7.5 14.5 5.5 15 4.5 16Z"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                        />
                                    </svg>
                                </span>
                            {/each}
                        </button>
                    {/each}
                </div>
            </div>

            {#if (mode === 'teams' ? teamScores.length : scores.length) === 0}
                <p class="text-sm text-slate-600 dark:text-slate-400">
                    {mode === 'teams' ? 'No team scores registered yet.' : 'No scores registered yet.'}
                </p>
            {/if}
        </div>
    {/if}
</div>
