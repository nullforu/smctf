<script lang="ts">
    import { api } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import type { ScoreEntry } from '../lib/types'
    import { navigate as _navigate } from '../lib/router'
    
    const navigate = _navigate

    let scores: ScoreEntry[] = $state([])
    let loading = $state(true)
    let errorMessage = $state('')

    const loadScoreboard = async () => {
        loading = true
        errorMessage = ''

        try {
            scores = await api.leaderboard()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    $effect(() => {
        loadScoreboard()
    })
</script>

<div
    class="min-w-0 rounded-2xl border border-slate-200 bg-white p-4 sm:p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
>
    <h3 class="text-lg text-slate-900 dark:text-slate-100">Leaderboard</h3>
    {#if loading}
        <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">Loading...</p>
    {:else if errorMessage}
        <p class="mt-4 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
    {:else}
        <div class="mt-4 space-y-3">
            {#each scores as entry, index}
                <button
                    class="w-full flex items-center justify-between rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 dark:border-slate-800/70 dark:bg-slate-950/40 cursor-pointer"
                    onclick={() => navigate(`/users/${entry.user_id}`)}
                >
                    <div class="flex min-w-0 items-center gap-3">
                        <span class="text-xs text-slate-500">#{index + 1}</span>
                        <span class="truncate text-sm text-slate-900 dark:text-slate-100">{entry.username}</span>
                    </div>
                    <span class="text-sm text-teal-600 dark:text-teal-200">{entry.score} pts</span>
                </button>
            {/each}
            {#if scores.length === 0}
                <p class="text-sm text-slate-600 dark:text-slate-400">No scores registered yet.</p>
            {/if}
        </div>
    {/if}
</div>
