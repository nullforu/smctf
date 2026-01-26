<script lang="ts">
    import { api } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import type { ScoreEntry } from '../lib/types'

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

<div class="min-w-0 rounded-2xl border border-slate-800/80 bg-slate-900/40 p-4 sm:p-6">
    <h3 class="text-lg text-slate-100">Leaderboard</h3>
    {#if loading}
        <p class="mt-4 text-sm text-slate-400">불러오는 중...</p>
    {:else if errorMessage}
        <p class="mt-4 text-sm text-rose-200">{errorMessage}</p>
    {:else}
        <div class="mt-4 space-y-3">
            {#each scores as entry, index}
                <div
                    class="flex items-center justify-between rounded-xl border border-slate-800/70 bg-slate-950/40 px-4 py-3"
                >
                    <div class="flex min-w-0 items-center gap-3">
                        <span class="text-xs text-slate-500">#{index + 1}</span>
                        <span class="truncate text-sm text-slate-100">{entry.username}</span>
                    </div>
                    <span class="text-sm text-teal-200">{entry.score} pts</span>
                </div>
            {/each}
            {#if scores.length === 0}
                <p class="text-sm text-slate-400">아직 점수가 등록되지 않았습니다.</p>
            {/if}
        </div>
    {/if}
</div>
