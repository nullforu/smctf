<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError, formatDateTime } from '../lib/utils'

    let limit = $state(50)
    let interval = $state(10)

    let scores: Array<{ user_id: number; username: string; score: number }> = $state([])
    let timeline: {
        interval_minutes: number
        users: Array<{ user_id: number; username: string; score: number }>
        buckets: Array<{
            bucket: string
            scores: Array<{ user_id: number; username: string; score: number }>
        }>
    } | null = $state(null)

    let loading = $state(true)
    let timelineLoading = $state(true)
    let errorMessage = $state('')
    let timelineError = $state('')

    const loadScoreboard = async () => {
        loading = true
        errorMessage = ''
        try {
            scores = await api.scoreboard(limit)
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const loadTimeline = async () => {
        timelineLoading = true
        timelineError = ''
        try {
            timeline = await api.timeline(interval, limit)
        } catch (error) {
            timelineError = formatApiError(error).message
        } finally {
            timelineLoading = false
        }
    }

    onMount(async () => {
        await loadScoreboard()
        await loadTimeline()
    })
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-100">Scoreboard</h2>
            <p class="mt-2 text-sm text-slate-400">상위 랭킹과 시간대별 점수 흐름을 확인하세요.</p>
        </div>
        <div class="flex flex-wrap gap-3 text-xs text-slate-300">
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Limit
                <input
                    class="w-16 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="1"
                    max="200"
                    bind:value={limit}
                    onchange={() => {
                        loadScoreboard()
                        loadTimeline()
                    }}
                />
            </label>
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Interval
                <input
                    class="w-16 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="1"
                    max="120"
                    bind:value={interval}
                    onchange={loadTimeline}
                />
                <span class="text-slate-400">min</span>
            </label>
        </div>
    </div>

    <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1.2fr]">
        <div class="rounded-2xl border border-slate-800/80 bg-slate-900/40 p-6">
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
                            <div class="flex items-center gap-3">
                                <span class="text-xs text-slate-500">#{index + 1}</span>
                                <span class="text-sm text-slate-100">{entry.username}</span>
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

        <div class="rounded-2xl border border-slate-800/80 bg-slate-900/40 p-6">
            <h3 class="text-lg text-slate-100">Timeline</h3>
            {#if timelineLoading}
                <p class="mt-4 text-sm text-slate-400">타임라인을 계산 중...</p>
            {:else if timelineError}
                <p class="mt-4 text-sm text-rose-200">{timelineError}</p>
            {:else if timeline}
                <p class="mt-2 text-xs text-slate-500">{timeline.interval_minutes}분 간격 기준</p>
                <div class="mt-4 space-y-4">
                    {#each timeline.buckets as bucket}
                        <div class="rounded-xl border border-slate-800/70 bg-slate-950/40 p-4">
                            <p class="text-xs text-slate-400">{formatDateTime(bucket.bucket)}</p>
                            <div class="mt-3 space-y-2">
                                {#each bucket.scores as score}
                                    <div class="flex items-center justify-between text-sm">
                                        <span class="text-slate-200">{score.username}</span>
                                        <span class="text-teal-200">{score.score} pts</span>
                                    </div>
                                {/each}
                                {#if bucket.scores.length === 0}
                                    <p class="text-xs text-slate-500">기록된 점수가 없습니다.</p>
                                {/if}
                            </div>
                        </div>
                    {/each}
                    {#if timeline.buckets.length === 0}
                        <p class="text-sm text-slate-400">타임라인 데이터가 없습니다.</p>
                    {/if}
                </div>
            {/if}
        </div>
    </div>
</section>
