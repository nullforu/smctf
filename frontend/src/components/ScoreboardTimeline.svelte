<script lang="ts">
    import { onDestroy, tick, untrack } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError, formatDateTime } from '../lib/utils'
    import {
        buildChartModel,
        chartLayout,
        chartUserLimit,
        type ChartSubmissionPoint,
        type ChartModel,
    } from '../routes/scoreboardChart'
    import type { TimelineSubmission, TimelineResponse } from '../lib/types'
    import { navigate } from '../lib/router'

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
    let chartModel: ChartModel | null = $state(null)
    let hoveredUserId: number | null = $state(null)
    let tooltip: TooltipState | null = $state(null)
    let chartContainer: HTMLDivElement | null = $state(null)
    let chartWidth = $state(chartLayout.width)
    let resizeObserver: ResizeObserver | null = null
    let tooltipBox: HTMLDivElement | null = $state(null)

    let loading = $state(true)
    let errorMessage = $state('')
    let requestId = $state(0)

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
        const nextWidth = Math.floor(chartContainer.clientWidth || chartLayout.width)
        const widthChanged = nextWidth !== chartWidth

        if (widthChanged) {
            chartWidth = nextWidth
            if (timeline) chartModel = buildChartModel(timeline, nextWidth)
        }

        if (!resizeObserver && typeof ResizeObserver !== 'undefined') {
            resizeObserver = new ResizeObserver(syncChartSize)
            resizeObserver.observe(chartContainer)
        }
    }

    const loadTimeline = async (modeValue: 'users' | 'teams') => {
        requestId += 1
        const currentRequest = requestId
        loading = true
        errorMessage = ''
        chartModel = null
        tooltip = null

        try {
            if (modeValue === 'teams') {
                const rawTeamTimeline = await api.timelineTeams()
                if (currentRequest !== requestId) return
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
                if (currentRequest !== requestId) return
            }
            chartModel = timeline ? buildChartModel(timeline, chartWidth) : null

            await tick()
            syncChartSize()
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
            loadTimeline(selectedMode)
        })
    })

    $effect(() => {
        if (!chartContainer) return
        syncChartSize()
    })

    onDestroy(() => {
        resizeObserver?.disconnect()
        resizeObserver = null
    })
</script>

<div class="min-w-0 rounded-2xl border border-border bg-surface p-4 sm:p-6">
    <h3 class="text-lg text-text">Timeline</h3>
    {#if loading}
        <p class="mt-4 text-sm text-text-muted">Calculating timeline...</p>
    {:else if errorMessage}
        <p class="mt-4 text-sm text-danger">{errorMessage}</p>
    {:else if timeline}
        <div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-text-muted">
            <span>
                Top {Math.min(chartUserLimit, chartModel?.series?.length || 0)}
                {mode === 'teams' ? 'teams' : 'users'}
            </span>
        </div>
        {#if chartModel}
            <div class="mt-4 rounded-xl border border-border bg-surface-muted p-4">
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
                                            class="stroke-border"
                                            stroke-width="1"
                                        />
                                        <text
                                            x={chartModel.padding.left - 10}
                                            y={tick.y + 4}
                                            text-anchor="end"
                                            fill="currentColor"
                                            style="font-size: 10px"
                                            class="text-text-subtle"
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
                                            class="stroke-border"
                                            stroke-width="1"
                                        />
                                        <text
                                            x={tick.x}
                                            y={chartModel.height - chartModel.padding.bottom + 18}
                                            text-anchor="middle"
                                            fill="currentColor"
                                            style="font-size: 10px"
                                            class="text-text-subtle"
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
                                            class="text-contrast-foreground"
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
                        class="pointer-events-none absolute z-10 w-60 max-w-[70vw] rounded-lg border border-border bg-surface/95 p-3 text-xs text-text shadow-xl"
                        bind:this={tooltipBox}
                        style={`left: ${tooltip?.left ?? 0}px; top: ${tooltip?.top ?? 0}px`}
                        class:hidden={!tooltip}
                    >
                        {#if tooltip}
                            <p class="text-text">
                                {mode === 'teams' ? 'Team' : 'User'}: {tooltip.username}
                            </p>
                            <p class="mt-1 text-sm text-text">
                                {tooltip.submission.challenge_count > 1
                                    ? `Solved ${tooltip.submission.challenge_count} challenges`
                                    : 'Challenge solved'}
                            </p>
                            <p class="mt-1 text-text-muted">
                                {formatDateTimeLocal(tooltip.submission.timestamp)}
                            </p>
                            <p class="mt-1 text-accent">+{tooltip.submission.points} pts</p>
                        {/if}
                    </div>
                </div>
                <div class="mt-3 flex flex-wrap gap-3 text-xs text-text-muted">
                    {#each chartModel.series as series}
                        {#if mode === 'teams'}
                            <button
                                class="flex items-center gap-2"
                                class:opacity-40={hoveredUserId && hoveredUserId !== series.user_id}
                                class:text-text={hoveredUserId === series.user_id}
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
                                class:text-text={hoveredUserId === series.user_id}
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
                <div class="mt-2 flex justify-between text-xs text-text-muted">
                    <span>{formatDateTimeLocal(chartModel.startLabel)}</span>
                    <span>{formatDateTimeLocal(chartModel.endLabel)}</span>
                </div>
            </div>
        {:else}
            <p class="mt-4 text-sm text-text-muted">No timeline data available.</p>
        {/if}
    {/if}
</div>
