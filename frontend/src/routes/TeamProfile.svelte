<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { TeamDetail, TeamMember, TeamSolvedChallenge } from '../lib/types'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import { navigate as _navigate } from '../lib/router'

    const navigate = _navigate

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let team: TeamDetail | null = $state(null)
    let members: TeamMember[] = $state([])
    let solved: TeamSolvedChallenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')

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

    $effect(() => {
        if (routeParams.id) {
            loadTeam(parseInt(routeParams.id))
        }
    })

    onMount(() => {
        if (routeParams.id) {
            loadTeam(parseInt(routeParams.id))
        }
    })
</script>

<section class="fade-in">
    <div class="mb-6">
        <button
            class="inline-flex items-center gap-2 text-sm text-slate-600 hover:text-teal-600 dark:text-slate-400 dark:hover:text-teal-400"
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
        <div class="rounded-2xl border border-slate-200 bg-white p-8 dark:border-slate-800/70 dark:bg-slate-900/40">
            <p class="text-center text-sm text-slate-600 dark:text-slate-400">Loading...</p>
        </div>
    {:else if errorMessage}
        <div class="rounded-2xl border border-rose-200 bg-rose-50 p-8 dark:border-rose-900/50 dark:bg-rose-950/20">
            <p class="text-center text-sm text-rose-700 dark:text-rose-300">{errorMessage}</p>
        </div>
    {:else if team}
        <div>
            <div class="flex flex-wrap items-end justify-between gap-4">
                <div>
                    <h2 class="text-3xl text-slate-900 dark:text-slate-100">{team.name}</h2>
                    <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">Team ID: {team.id}</p>
                </div>
                <div class="flex flex-wrap gap-2 text-xs">
                    <span
                        class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-slate-700 dark:border-slate-800/70 dark:bg-slate-900/60 dark:text-slate-300"
                    >
                        Members: {team.member_count}
                    </span>
                    <span
                        class="rounded-full border border-teal-200 bg-teal-50 px-3 py-1 text-teal-700 dark:border-teal-800/40 dark:bg-teal-900/20 dark:text-teal-200"
                    >
                        Total Score: {team.total_score} pts
                    </span>
                </div>
            </div>

            <div class="mt-8 grid gap-6 lg:grid-cols-[1.4fr_1fr]">
                <div
                    class="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg text-slate-900 dark:text-slate-100">Members</h3>
                        <span class="text-xs text-slate-500 dark:text-slate-400">{members.length} total</span>
                    </div>

                    {#if members.length === 0}
                        <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">No members registered.</p>
                    {:else}
                        <div class="mt-4 overflow-x-auto">
                            <table class="w-full pl-4 text-left text-sm text-slate-700 dark:text-slate-300">
                                <thead class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                                    <tr>
                                        <th class="py-2 px-4">ID</th>
                                        <th class="py-2 pr-4">Username</th>
                                        <th class="py-2">Role</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each members as member}
                                        <tr
                                            class="border-t border-slate-200/70 dark:border-slate-800/70 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-900/60"
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

                <div
                    class="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg text-slate-900 dark:text-slate-100">Solved Challenges</h3>
                        <span class="text-xs text-slate-500 dark:text-slate-400">{solved.length} total</span>
                    </div>

                    {#if solved.length === 0}
                        <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">No challenges solved yet.</p>
                    {:else}
                        <div class="mt-4 space-y-3">
                            {#each solved as entry}
                                <div
                                    class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40"
                                >
                                    <div class="flex items-center justify-between gap-3">
                                        <div>
                                            <p class="text-sm text-slate-900 dark:text-slate-100">{entry.title}</p>
                                            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                                                Last solved: {formatDateTimeLocal(entry.last_solved_at)}
                                            </p>
                                        </div>
                                        <div class="text-right">
                                            <p class="text-sm text-teal-600 dark:text-teal-200">{entry.points} pts</p>
                                            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
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
