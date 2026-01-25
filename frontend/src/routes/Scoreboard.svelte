<script lang="ts">
    import { onDestroy, onMount, tick } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import { buildChartModel, chartLayout, type ChartEventPoint, type ChartModel } from './scoreboardChart'
    import type { ScoreEntry, TimelineEvent, TimelineResponse } from '../lib/types'

    let limit = $state(50)
    let interval = $state(10)
    let windowMinutes = $state(60)

    interface TooltipState {
        left: number
        top: number
        event: TimelineEvent
        username: string
    }

    let scores: ScoreEntry[] = $state([])
    let timeline: TimelineResponse | null = $state(null)
    let chartModel: ChartModel | null = $state(null)
    let hoveredUserId: number | null = $state(null)
    let tooltip: TooltipState | null = $state(null)
    let chartContainer: HTMLDivElement | null = $state(null)
    let chartWidth = $state(chartLayout.width)
    let resizeObserver: ResizeObserver | null = null
    let tooltipBox: HTMLDivElement | null = $state(null)

    const chartUserLimit = 10

    let loading = $state(true)
    let timelineLoading = $state(true)
    let errorMessage = $state('')
    let timelineError = $state('')

    const formatDateTimeSafe = formatDateTime

    const showTooltip = (event: MouseEvent, point: ChartEventPoint, username: string) => {
        if (!chartContainer) return
        const rect = chartContainer.getBoundingClientRect()

        const tooltipWidth = tooltipBox?.offsetWidth || 240
        const tooltipHeight = tooltipBox?.offsetHeight || 120

        const padding = 12

        const rawLeft = event.clientX - rect.left + padding
        const rawTop = event.clientY - rect.top + padding

        const maxLeft = rect.width - tooltipWidth - padding
        const maxTop = rect.height - tooltipHeight - padding

        tooltip = {
            left: Math.max(padding, Math.min(rawLeft, maxLeft)),
            top: Math.max(padding, Math.min(rawTop, maxTop)),
            event: point.event,
            username,
        }
    }

    const clearTooltip: () => void = () => (tooltip = null)

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

    const syncChartSize = () => {
        if (!chartContainer) return
        chartWidth = Math.floor(chartContainer.clientWidth || chartLayout.width)

        if (timeline) chartModel = buildChartModel(timeline, windowMinutes, chartWidth)

        if (!resizeObserver && typeof ResizeObserver !== 'undefined') {
            resizeObserver = new ResizeObserver(syncChartSize)
            resizeObserver.observe(chartContainer)
        }
    }

    const loadTimeline = async () => {
        timelineLoading = true
        timelineError = ''
        chartModel = null
        tooltip = null

        try {
            timeline = await api.timeline(interval, limit, windowMinutes)
            chartModel = timeline ? buildChartModel(timeline, windowMinutes, chartWidth) : null

            await tick()
            syncChartSize()
        } catch (error) {
            timelineError = formatApiError(error).message
        } finally {
            timelineLoading = false
        }
    }

    onMount(async () => {
        await loadScoreboard()
        await loadTimeline()
        syncChartSize()
    })

    onDestroy(() => {
        resizeObserver?.disconnect()
        resizeObserver = null
    })
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-100">Scoreboard</h2>
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
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Window
                <input
                    class="w-20 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="10"
                    max="1440"
                    bind:value={windowMinutes}
                    onchange={loadTimeline}
                />
                <span class="text-slate-400">min</span>
            </label>
        </div>
    </div>

    <div class="mt-6 grid min-w-0 grid-cols-1 gap-6">
        <div class="min-w-0 rounded-2xl border border-slate-800/80 bg-slate-900/40 p-4 sm:p-6">
            <h3 class="text-lg text-slate-100">Timeline</h3>
            {#if timelineLoading}
                <p class="mt-4 text-sm text-slate-400">타임라인을 계산 중...</p>
            {:else if timelineError}
                <p class="mt-4 text-sm text-rose-200">{timelineError}</p>
            {:else if timeline}
                <div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-500">
                    <span>최근 {windowMinutes}분</span>
                    <span>·</span>
                    <span>{timeline.interval_minutes}분 간격</span>
                    <span>·</span>
                    <span>상위 {Math.min(chartUserLimit, timeline.users.length)}명</span>
                </div>
                {#if chartModel}
                    <div class="mt-4 rounded-xl border border-slate-800/70 bg-slate-950/40 p-4">
                        <div
                            class="relative min-w-0 w-full overflow-hidden"
                            bind:this={chartContainer}
                            role="group"
                            aria-label="score timeline chart"
                            onmouseleave={() => {
                                hoveredUserId = null
                                clearTooltip()
                            }}
                        >
                            <div class="overflow-x-auto overflow-y-hidden overscroll-x-contain touch-pan-x">
                                <div class="w-full" style={`width: ${chartModel.width}px`}>
                                    <svg
                                        class="block h-72 w-full"
                                        viewBox={`0 0 ${chartModel.width} ${chartModel.height}`}
                                        role="img"
                                        aria-label="score timeline chart"
                                    >
                                        <rect
                                            x="0"
                                            y="0"
                                            width={chartModel.width}
                                            height={chartModel.height}
                                            fill="transparent"
                                        />
                                        <g>
                                            {#each chartModel.ticks as tick}
                                                <line
                                                    x1={chartModel.padding.left}
                                                    x2={chartModel.width - chartModel.padding.right}
                                                    y1={tick.y}
                                                    y2={tick.y}
                                                    class="stroke-slate-800/60"
                                                    stroke-width="1"
                                                />
                                                <text
                                                    x={chartModel.padding.left - 10}
                                                    y={tick.y + 4}
                                                    text-anchor="end"
                                                    fill="currentColor"
                                                    style="font-size: 10px"
                                                    class="text-slate-500"
                                                >
                                                    {tick.value}
                                                </text>
                                            {/each}
                                        </g>
                                        <g>
                                            {#each chartModel.timeTicks as tick}
                                                <line
                                                    x1={tick.x}
                                                    x2={tick.x}
                                                    y1={chartModel.height - chartModel.padding.bottom}
                                                    y2={chartModel.height - chartModel.padding.bottom + 6}
                                                    class="stroke-slate-800/70"
                                                    stroke-width="1"
                                                />
                                                <text
                                                    x={tick.x}
                                                    y={chartModel.height - chartModel.padding.bottom + 18}
                                                    text-anchor="middle"
                                                    fill="currentColor"
                                                    style="font-size: 10px"
                                                    class="text-slate-500"
                                                >
                                                    {tick.label}
                                                </text>
                                            {/each}
                                        </g>
                                        {#each chartModel.series as series}
                                            <path
                                                d={series.path}
                                                fill="none"
                                                stroke={series.color}
                                                stroke-width={hoveredUserId === series.user_id ? 3 : 2}
                                                stroke-linecap="round"
                                                stroke-linejoin="round"
                                                class:opacity-30={hoveredUserId && hoveredUserId !== series.user_id}
                                                role="presentation"
                                                aria-hidden="true"
                                                onmouseenter={() => {
                                                    hoveredUserId = series.user_id
                                                }}
                                                onmouseleave={() => {
                                                    hoveredUserId = null
                                                }}
                                            />
                                        {/each}
                                        {#each chartModel.series as series}
                                            {#each series.eventPoints as point}
                                                <circle
                                                    cx={point.x}
                                                    cy={point.y}
                                                    r={hoveredUserId === series.user_id ? 5.5 : 4}
                                                    fill={series.color}
                                                    stroke="rgba(15, 23, 42, 0.9)"
                                                    stroke-width="1.4"
                                                    class:opacity-30={hoveredUserId && hoveredUserId !== series.user_id}
                                                    role="presentation"
                                                    aria-hidden="true"
                                                    onmouseenter={(event) => {
                                                        hoveredUserId = series.user_id
                                                        showTooltip(event, point, series.username)
                                                    }}
                                                    onmousemove={(event) => {
                                                        showTooltip(event, point, series.username)
                                                    }}
                                                    onmouseleave={() => {
                                                        hoveredUserId = null
                                                        clearTooltip()
                                                    }}
                                                />
                                            {/each}
                                        {/each}
                                    </svg>
                                </div>
                            </div>
                            <div
                                class="pointer-events-none absolute z-10 w-60 max-w-[70vw] rounded-lg border border-slate-700/80 bg-slate-950/90 p-3 text-xs text-slate-200 shadow-xl"
                                bind:this={tooltipBox}
                                style={`left: ${tooltip?.left ?? 0}px; top: ${tooltip?.top ?? 0}px`}
                                class:hidden={!tooltip}
                            >
                                {#if tooltip}
                                    <p class="text-slate-300">{tooltip.username}</p>
                                    <p class="mt-1 text-sm text-slate-100">{tooltip.event.challenge_title}</p>
                                    <p class="mt-1 text-slate-400">
                                        {formatDateTimeSafe(tooltip.event.submitted_at)}
                                    </p>
                                    <p class="mt-1 text-teal-200">+{tooltip.event.points} pts</p>
                                {/if}
                            </div>
                        </div>
                        <div class="mt-3 flex flex-wrap gap-3 text-xs text-slate-400">
                            {#each chartModel.series as series}
                                <span
                                    class="flex items-center gap-2 transition"
                                    class:opacity-40={hoveredUserId && hoveredUserId !== series.user_id}
                                    class:text-slate-100={hoveredUserId === series.user_id}
                                    role="button"
                                    tabindex="0"
                                    aria-label={`${series.username} highlight`}
                                    onmouseenter={() => {
                                        hoveredUserId = series.user_id
                                    }}
                                    onmouseleave={() => {
                                        hoveredUserId = null
                                    }}
                                >
                                    <span class="h-2 w-2 rounded-full" style={`background-color: ${series.color}`}
                                    ></span>
                                    {series.username}
                                </span>
                            {/each}
                        </div>
                        <div class="mt-2 flex justify-between text-xs text-slate-500">
                            <span>{formatDateTimeSafe(chartModel.startLabel)}</span>
                            <span>{formatDateTimeSafe(chartModel.endLabel)}</span>
                        </div>
                    </div>
                {:else}
                    <p class="mt-4 text-sm text-slate-400">타임라인 데이터가 없습니다.</p>
                {/if}
            {/if}
        </div>
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
    </div>
</section>
