<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { UserDetail, SolvedChallenge } from '../lib/types'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import { navigate as _navigate } from '../lib/router'

    const navigate = _navigate

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let user: UserDetail | null = $state(null)
    let solved: SolvedChallenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')

    const formatDateTimeLocal = formatDateTime

    const loadUserProfile = async (userId: string) => {
        loading = true
        errorMessage = ''
        user = null
        solved = []

        try {
            const [userDetail, solvedData] = await Promise.all([
                api.user(parseInt(userId)),
                api.userSolved(parseInt(userId)),
            ])
            user = userDetail
            solved = solvedData
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    $effect(() => {
        if (routeParams.id) {
            loadUserProfile(routeParams.id)
        }
    })

    onMount(() => {
        if (routeParams.id) {
            loadUserProfile(routeParams.id)
        }
    })
</script>

<section class="fade-in">
    <div class="mb-6">
        <button
            class="inline-flex items-center gap-2 text-sm text-slate-600 hover:text-teal-600 dark:text-slate-400 dark:hover:text-teal-400"
            onclick={() => navigate('/users')}
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
            Back to Users
        </button>
    </div>

    {#if loading}
        <div class="rounded-2xl border border-slate-200 bg-white p-8 dark:border-slate-800/70 dark:bg-slate-900/40">
            <p class="text-center text-sm text-slate-600 dark:text-slate-400">불러오는 중...</p>
        </div>
    {:else if errorMessage}
        <div class="rounded-2xl border border-rose-200 bg-rose-50 p-8 dark:border-rose-900/50 dark:bg-rose-950/20">
            <p class="text-center text-sm text-rose-700 dark:text-rose-300">{errorMessage}</p>
        </div>
    {:else if user}
        <div>
            <div class="flex flex-wrap items-end justify-between gap-4">
                <div>
                    <h2 class="text-3xl text-slate-900 dark:text-slate-100">{user.username}</h2>
                    <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">User ID: {user.id}</p>
                </div>
                <div>
                    <span
                        class="inline-flex items-center rounded-full px-3 py-1 text-sm font-medium uppercase {user.role ===
                        'admin'
                            ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
                            : 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300'}"
                    >
                        {user.role}
                    </span>
                </div>
            </div>

            <div class="mt-8">
                <div
                    class="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg text-slate-900 dark:text-slate-100">Solved Challenges</h3>
                        <span class="text-sm text-slate-600 dark:text-slate-400">
                            {solved.length}
                            {solved.length === 1 ? 'problem' : 'problems'}
                        </span>
                    </div>

                    <div class="mt-6 space-y-3">
                        {#each solved as item}
                            <div
                                class="rounded-xl border border-slate-200 bg-slate-50 p-5 dark:border-slate-800/70 dark:bg-slate-950/40"
                            >
                                <div class="flex items-start justify-between">
                                    <div class="flex-1">
                                        <div class="flex items-center gap-3">
                                            <h4 class="text-base font-medium text-slate-900 dark:text-slate-100">
                                                {item.title}
                                            </h4>
                                            <span
                                                class="inline-flex items-center rounded-full bg-teal-100 px-2.5 py-0.5 text-xs font-medium text-teal-800 dark:bg-teal-900/30 dark:text-teal-300"
                                            >
                                                {item.points} pts
                                            </span>
                                        </div>
                                        <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">
                                            Solved at {formatDateTimeLocal(item.solved_at)}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        {/each}
                        {#if solved.length === 0}
                            <div
                                class="rounded-xl border border-slate-200 bg-slate-50 p-8 text-center dark:border-slate-800/70 dark:bg-slate-950/40"
                            >
                                <p class="text-sm text-slate-600 dark:text-slate-400">아직 해결한 문제가 없습니다.</p>
                            </div>
                        {/if}
                    </div>
                </div>
            </div>

            {#if solved.length > 0}
                <div
                    class="mt-8 rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <h3 class="text-lg text-slate-900 dark:text-slate-100">Statistics</h3>
                    <div class="mt-4 grid gap-4 sm:grid-cols-2">
                        <div
                            class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40"
                        >
                            <p class="text-xs text-slate-600 dark:text-slate-400">Total Points</p>
                            <p class="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">
                                {solved.reduce((sum, s) => sum + s.points, 0)}
                            </p>
                        </div>
                        <div
                            class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40"
                        >
                            <p class="text-xs text-slate-600 dark:text-slate-400">Problems Solved</p>
                            <p class="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">
                                {solved.length}
                            </p>
                        </div>
                    </div>
                </div>
            {/if}
        </div>
    {/if}
</section>
