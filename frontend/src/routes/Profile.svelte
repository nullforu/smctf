<script lang="ts">
    import { onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api } from '../lib/api'
    import type { SolvedChallenge } from '../lib/types'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import { navigate } from '../lib/router'

    let solved: SolvedChallenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let auth = $state(get(authStore))

    const formatDateTimeLocal = formatDateTime

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    const loadSolved = async () => {
        if (!auth.user) return

        loading = true
        errorMessage = ''

        try {
            solved = await api.solved()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    onMount(loadSolved)
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Profile</h2>
        </div>
    </div>

    {#if !auth.user}
        <div
            class="mt-6 rounded-2xl border border-amber-500/40 bg-amber-500/10 p-6 text-sm text-amber-800 dark:text-amber-100"
        >
            <a class="underline" href="/login" onclick={(e) => navigate('/login', e)}>로그인</a> 후 프로필을 확인할 수 있습니다.
        </div>
    {:else}
        <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1.3fr]">
            <div class="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
                <h3 class="text-lg text-slate-900 dark:text-slate-100">계정</h3>
                <div class="mt-4 space-y-2 text-sm text-slate-700 dark:text-slate-300">
                    <div class="flex justify-between">
                        <span class="text-slate-600 dark:text-slate-400">Username</span>
                        <span>{auth.user.username}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-slate-600 dark:text-slate-400">Email</span>
                        <span>{auth.user.email}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-slate-600 dark:text-slate-400">Role</span>
                        <span class="uppercase text-teal-600 dark:text-teal-200">{auth.user.role}</span>
                    </div>
                </div>
            </div>

            <div class="rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
                <div class="flex items-center justify-between">
                    <h3 class="text-lg text-slate-900 dark:text-slate-100">Solved</h3>
                    <button
                        class="rounded-full border border-slate-300 px-3 py-1 text-xs text-slate-700 hover:border-teal-500 dark:border-slate-700 dark:text-slate-300 dark:hover:border-teal-400"
                        onclick={loadSolved}
                    >
                        새로고침
                    </button>
                </div>
                {#if loading}
                    <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">불러오는 중...</p>
                {:else if errorMessage}
                    <p class="mt-4 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
                {:else}
                    <div class="mt-4 space-y-3">
                        {#each solved as item}
                            <div
                                class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40"
                            >
                                <div class="flex items-center justify-between">
                                    <span class="text-sm text-slate-900 dark:text-slate-100">{item.title}</span>
                                    <span class="text-xs text-teal-600 dark:text-teal-200">{item.points} pts</span>
                                </div>
                                <p class="mt-2 text-xs text-slate-600 dark:text-slate-400">
                                    {formatDateTimeLocal(item.solved_at)}
                                </p>
                            </div>
                        {/each}
                        {#if solved.length === 0}
                            <p class="text-sm text-slate-600 dark:text-slate-400">아직 해결한 문제가 없습니다.</p>
                        {/if}
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</section>
