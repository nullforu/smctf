<script lang="ts">
    import { onDestroy, onMount, tick } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import { buildChartModel, chartLayout, type ChartSubmissionPoint, type ChartModel } from '../routes/scoreboardChart'
    import type { TeamTimelineResponse, TimelineSubmission, TimelineResponse } from '../lib/types'
    import { navigate as _navigate } from '../lib/router'

    const navigate = _navigate

    interface Props {
        mode?: 'users' | 'teams'
    }

    interface TooltipState {
        left: number
        top: number
        submission: TimelineSubmission
        username: string
    }

    let { mode = 'users' }: Props = $props()

    let timeline: TimelineResponse | null = $state(null)
    let rawTeamTimeline: TeamTimelineResponse | null = $state(null)
    let chartModel: ChartModel | null = $state(null)
    let hoveredUserId: number | null = $state(null)
    let tooltip: TooltipState | null = $state(null)
    let chartContainer: HTMLDivElement | null = $state(null)
    let chartWidth = $state(chartLayout.width)
    let resizeObserver: ResizeObserver | null = null
    let tooltipBox: HTMLDivElement | null = $state(null)

    const chartUserLimit = 10

    let loading = $state(true)
    let errorMessage = $state('')

    const formatDateTimeLocal = formatDateTime

    const showTooltip = (event: MouseEvent, point: ChartSubmissionPoint, username: string) => {
        if (!chartContainer || !tooltipBox) return

        const rect = chartContainer.getBoundingClientRect()
        const tooltipWidth = tooltipBox.offsetWidth
        const tooltipHeight = tooltipBox.offsetHeight
        const padding = 12

        const rawLeft = event.clientX - rect.left + padding
        const rawTop = event.clientY - rect.top + padding
        const maxLeft = rect.width - tooltipWidth - padding
        const maxTop = rect.height - tooltipHeight - padding

        tooltip = {
            left: Math.max(padding, Math.min(rawLeft, maxLeft)),
            top: Math.max(padding, Math.min(rawTop, maxTop)),
            submission: point.submission,
            username,
        }
    }

    const clearTooltip = () => {
        tooltip = null
    }

    const syncChartSize = () => {
        if (!chartContainer) return
        chartWidth = Math.floor(chartContainer.clientWidth || chartLayout.width)

        if (timeline) chartModel = buildChartModel(timeline, chartWidth)

        if (!resizeObserver && typeof ResizeObserver !== 'undefined') {
            resizeObserver = new ResizeObserver(syncChartSize)
            resizeObserver.observe(chartContainer)
        }
    }

    const loadTimeline = async () => {
        loading = true
        errorMessage = ''
        chartModel = null
        tooltip = null

        try {
            if (mode === 'teams') {
                rawTeamTimeline = await api.timelineTeams()
                timeline = rawTeamTimeline
                    ? {
                          submissions: rawTeamTimeline.submissions.map((sub) => ({
                              timestamp: sub.timestamp,
                              user_id: sub.team_id,
                              username: sub.team_name,
                              points: sub.points,
                              challenge_count: sub.challenge_count,
                          })),
                      }
                    : null
            } else {
                timeline = await api.timeline()
                rawTeamTimeline = null
            }
            chartModel = timeline ? buildChartModel(timeline, chartWidth) : null

            await tick()
            syncChartSize()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    onMount(async () => {
        await loadTimeline()
        syncChartSize()
    })

    onDestroy(() => {
        resizeObserver?.disconnect()
        resizeObserver = null
    })
</script>

<div
    class="min-w-0 rounded-2xl border border-slate-200 bg-white p-4 sm:p-6 dark:border-slate-800/80 dark:bg-slate-900/40"
>
    <h3 class="text-lg text-slate-900 dark:text-slate-100">Timeline</h3>
    {#if loading}
        <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">Calculating timeline...</p>
    {:else if errorMessage}
        <p class="mt-4 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
    {:else if timeline}
        <div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-slate-600 dark:text-slate-500">
            <span>
                Top {Math.min(chartUserLimit, chartModel?.series?.length || 0)}
                {mode === 'teams' ? 'teams' : 'users'}
            </span>
        </div>
        {#if chartModel}
            <div
                class="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40"
            >
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
                                            class="stroke-slate-300 dark:stroke-slate-800/60"
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
                                            class="stroke-slate-300 dark:stroke-slate-800/70"
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
                                    {#each series.submissionPoints as point}
                                        <circle
                                            cx={point.x}
                                            cy={point.y}
                                            r={hoveredUserId === series.user_id ? 5.5 : 4}
                                            fill={series.color}
                                            stroke="currentColor"
                                            stroke-width="1.4"
                                            class="text-white dark:text-slate-950"
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
                        class="pointer-events-none absolute z-10 w-60 max-w-[70vw] rounded-lg border border-slate-300 bg-white/95 p-3 text-xs text-slate-800 shadow-xl dark:border-slate-700/80 dark:bg-slate-950/90 dark:text-slate-200"
                        bind:this={tooltipBox}
                        style={`left: ${tooltip?.left ?? 0}px; top: ${tooltip?.top ?? 0}px`}
                        class:hidden={!tooltip}
                    >
                        {#if tooltip}
                            <p class="text-slate-700 dark:text-slate-300">
                                {mode === 'teams' ? 'Team' : 'User'}: {tooltip.username}
                            </p>
                            <p class="mt-1 text-sm text-slate-900 dark:text-slate-100">
                                {tooltip.submission.challenge_count > 1
                                    ? `Solved ${tooltip.submission.challenge_count} challenges`
                                    : 'Challenge solved'}
                            </p>
                            <p class="mt-1 text-slate-600 dark:text-slate-400">
                                {formatDateTimeLocal(tooltip.submission.timestamp)}
                            </p>
                            <p class="mt-1 text-teal-600 dark:text-teal-200">+{tooltip.submission.points} pts</p>
                        {/if}
                    </div>
                </div>
                <div class="mt-3 flex flex-wrap gap-3 text-xs text-slate-600 dark:text-slate-400">
                    {#each chartModel.series as series}
                        {#if mode === 'teams'}
                            <button
                                class="flex items-center gap-2"
                                class:opacity-40={hoveredUserId && hoveredUserId !== series.user_id}
                                class:text-slate-900={hoveredUserId === series.user_id}
                                class:dark:text-slate-100={hoveredUserId === series.user_id}
                                aria-label={`${series.username} highlight`}
                                onmouseenter={() => {
                                    hoveredUserId = series.user_id
                                }}
                                onmouseleave={() => {
                                    hoveredUserId = null
                                }}
                            >
                                <span class="h-2 w-2 rounded-full" style={`background-color: ${series.color}`}></span>
                                {series.username}
                            </button>
                        {:else}
                            <button
                                class="flex items-center gap-2 transition"
                                class:opacity-40={hoveredUserId && hoveredUserId !== series.user_id}
                                class:text-slate-900={hoveredUserId === series.user_id}
                                class:dark:text-slate-100={hoveredUserId === series.user_id}
                                tabindex="0"
                                aria-label={`${series.username} highlight`}
                                onmouseenter={() => {
                                    hoveredUserId = series.user_id
                                }}
                                onmouseleave={() => {
                                    hoveredUserId = null
                                }}
                                onclick={() => navigate(`/users/${series.user_id}`)}
                            >
                                <span class="h-2 w-2 rounded-full" style={`background-color: ${series.color}`}></span>
                                {series.username}
                            </button>
                        {/if}
                    {/each}
                </div>
                <div class="mt-2 flex justify-between text-xs text-slate-600 dark:text-slate-500">
                    <span>{formatDateTimeLocal(chartModel.startLabel)}</span>
                    <span>{formatDateTimeLocal(chartModel.endLabel)}</span>
                </div>
            </div>
        {:else}
            <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">No timeline data available.</p>
        {/if}
    {/if}
</div>
