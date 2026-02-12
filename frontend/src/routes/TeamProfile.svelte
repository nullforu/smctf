<script lang="ts">
    import { api } from '../lib/api'
    import type { TeamDetail, TeamMember, TeamSolvedChallenge } from '../lib/types'
    import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
    import { navigate } from '../lib/router'

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let team: TeamDetail | null = $state(null)
    let members: TeamMember[] = $state([])
    let solved: TeamSolvedChallenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let lastLoadedTeamId = $state<number | null>(null)

    const formatDateTimeLocal = formatDateTime

    const loadTeam = async (teamId: number) => {
        loading = true
        errorMessage = ''
        team = null
        members = []
        solved = []

        try {
            const [teamDetail, memberRows, solvedRows] = await Promise.all([
                api.teamDetail(teamId),
                api.teamMembers(teamId),
                api.teamSolved(teamId),
            ])
            team = teamDetail
            members = memberRows
            solved = solvedRows
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const routeTeamId = $derived(parseRouteId(routeParams.id))

    $effect(() => {
        if (routeTeamId === null) return
        if (lastLoadedTeamId === routeTeamId) return

        lastLoadedTeamId = routeTeamId
        loadTeam(routeTeamId)
    })
</script>

<section class="fade-in">
    <div class="mb-6">
        <button
            class="inline-flex items-center gap-2 text-sm text-text-muted hover:text-accent"
            onclick={() => navigate('/teams')}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                <path d="m15 18-6-6 6-6" />
            </svg>
            Back to Teams
        </button>
    </div>

    {#if loading}
        <div class="rounded-2xl border border-border bg-surface p-8">
            <p class="text-center text-sm text-text-muted">Loading...</p>
        </div>
    {:else if errorMessage}
        <div class="rounded-2xl border border-danger/30 bg-danger/10 p-8">
            <p class="text-center text-sm text-danger">{errorMessage}</p>
        </div>
    {:else if team}
        <div>
            <div class="flex flex-wrap items-end justify-between gap-4">
                <div>
                    <h2 class="text-3xl text-text">{team.name}</h2>
                    <p class="mt-1 text-sm text-text-muted">Team ID: {team.id}</p>
                </div>
                <div class="flex flex-wrap gap-2 text-xs">
                    <span class="rounded-full border border-border bg-surface-muted px-3 py-1 text-text">
                        Members: {team.member_count}
                    </span>
                    <span class="rounded-full border border-accent/30 bg-accent/10 px-3 py-1 text-accent-strong">
                        Total Score: {team.total_score} pts
                    </span>
                </div>
            </div>

            <div class="mt-8 grid gap-6 lg:grid-cols-[1.4fr_1fr]">
                <div class="rounded-2xl border border-border bg-surface p-6">
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg text-text">Members</h3>
                        <span class="text-xs text-text-subtle">{members.length} total</span>
                    </div>

                    {#if members.length === 0}
                        <p class="mt-4 text-sm text-text-subtle">No members registered.</p>
                    {:else}
                        <div class="mt-4 overflow-x-auto">
                            <table class="w-full pl-4 text-left text-sm text-text">
                                <thead class="text-xs uppercase tracking-wide text-text-subtle">
                                    <tr>
                                        <th class="py-2 px-4">ID</th>
                                        <th class="py-2 pr-4">Username</th>
                                        <th class="py-2">Role</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each members as member}
                                        <tr
                                            class="border-t border-border/70 cursor-pointer hover:bg-surface-muted"
                                            onclick={() => navigate(`/users/${member.id}`)}
                                        >
                                            <td class="py-3 px-4">{member.id}</td>
                                            <td class="py-3 pr-4">{member.username}</td>
                                            <td class="py-3">{member.role}</td>
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                    {/if}
                </div>

                <div class="rounded-2xl border border-border bg-surface p-6">
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg text-text">Solved Challenges</h3>
                        <span class="text-xs text-text-subtle">{solved.length} total</span>
                    </div>

                    {#if solved.length === 0}
                        <p class="mt-4 text-sm text-text-subtle">No challenges solved yet.</p>
                    {:else}
                        <div class="mt-4 space-y-3">
                            {#each solved as entry}
                                <div class="rounded-xl border border-border bg-surface-muted p-4">
                                    <div class="flex items-center justify-between gap-3">
                                        <div>
                                            <p class="text-sm text-text">{entry.title}</p>
                                            <p class="mt-1 text-xs text-text-subtle">
                                                Last solved: {formatDateTimeLocal(entry.last_solved_at)}
                                            </p>
                                        </div>
                                        <div class="text-right">
                                            <p class="text-sm text-accent">{entry.points} pts</p>
                                            <p class="mt-1 text-xs text-text-subtle">
                                                {entry.solve_count} solves
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</section>
