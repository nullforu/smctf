import type { TimelineEvent, TimelineResponse } from '../lib/types'

export interface ChartPoint {
    x: number
    y: number
    value: number
}

export interface ChartEventPoint {
    x: number
    y: number
    value: number
    event: TimelineEvent
}

export interface ChartAxisTick {
    value: number
    y: number
}

export interface ChartTimeTick {
    x: number
    label: string
}

export interface ChartPadding {
    top: number
    right: number
    bottom: number
    left: number
}

export interface ChartSeries {
    user_id: number
    username: string
    color: string
    path: string
    points: ChartPoint[]
    eventPoints: ChartEventPoint[]
}

export interface ChartModel {
    width: number
    height: number
    padding: ChartPadding
    ticks: ChartAxisTick[]
    timeTicks: ChartTimeTick[]
    series: ChartSeries[]
    startLabel: string
    endLabel: string
}

export const chartPalette = [
    '#38bdf8',
    '#34d399',
    '#fbbf24',
    '#f472b6',
    '#a78bfa',
    '#f97316',
    '#22d3ee',
    '#f87171',
    '#4ade80',
    '#60a5fa',
]

export const chartUserLimit = 10

export const chartLayout = {
    width: 720,
    height: 320,
    padding: { top: 20, right: 24, bottom: 36, left: 48 } as ChartPadding,
    ticks: 4,
}

const formatTime = (value: string) => {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value

    return date.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
}

export const buildChartModel = (
    data: TimelineResponse,
    windowMinutesValue: number,
    widthValue: number,
): ChartModel | null => {
    const baseWidth = Math.floor(widthValue || chartLayout.width)
    const resolvedWidth = Math.max(chartLayout.width, baseWidth)

    const users = data.users.slice(0, Math.min(chartUserLimit, data.users.length))

    if (users.length === 0) return null

    const now = Date.now()
    const safeWindowMinutes = Number.isFinite(windowMinutesValue) && windowMinutesValue > 0 ? windowMinutesValue : 60
    const windowStart = now - safeWindowMinutes * 60 * 1000
    const windowEnd = now
    const span = Math.max(1, windowEnd - windowStart)

    const plotWidth = resolvedWidth - chartLayout.padding.left - chartLayout.padding.right
    const plotHeight = chartLayout.height - chartLayout.padding.top - chartLayout.padding.bottom

    const events = (data.events || [])
        .map((event) => ({ event, time: new Date(event.submitted_at).getTime() }))
        .filter((entry) => !Number.isNaN(entry.time) && entry.time >= windowStart && entry.time <= windowEnd)
        .sort((a, b) => a.time - b.time)

    const eventsByUser = new Map<number, TimelineEvent[]>()
    for (const user of users) {
        eventsByUser.set(user.user_id, [])
    }
    for (const entry of events) {
        if (eventsByUser.has(entry.event.user_id)) {
            eventsByUser.get(entry.event.user_id)?.push(entry.event)
        }
    }

    let maxValue = 0
    for (const userEvents of eventsByUser.values()) {
        const total = userEvents.reduce((sum, ev) => sum + ev.points, 0)
        if (total > maxValue) maxValue = total
    }
    const safeMax = Math.max(1, maxValue)

    const xScale = (time: number) => chartLayout.padding.left + ((time - windowStart) / span) * plotWidth
    const yScale = (value: number) => chartLayout.padding.top + plotHeight - (value / safeMax) * plotHeight

    const series = users.map((user, index) => {
        const userEvents = eventsByUser.get(user.user_id) || []
        const eventPoints: ChartEventPoint[] = []
        let runningScore = 0

        for (const event of userEvents) {
            const time = new Date(event.submitted_at).getTime()
            const clampedTime = Math.min(windowEnd, Math.max(windowStart, time))
            runningScore += event.points
            eventPoints.push({
                event,
                value: runningScore,
                x: xScale(clampedTime),
                y: yScale(runningScore),
            })
        }

        const points: ChartPoint[] = [
            { x: xScale(windowStart), y: yScale(0), value: 0 },
            ...eventPoints.map((point) => ({ x: point.x, y: point.y, value: point.value })),
        ]

        const path = points
            .map((point, idx) => `${idx === 0 ? 'M' : 'L'}${point.x.toFixed(1)} ${point.y.toFixed(1)}`)
            .join(' ')

        return {
            user_id: user.user_id,
            username: user.username,
            color: chartPalette[index % chartPalette.length],
            path,
            points,
            eventPoints,
        }
    })

    const ticks = Array.from({ length: chartLayout.ticks + 1 }, (_, idx) => {
        const value = Math.round((safeMax / chartLayout.ticks) * idx)
        return { value, y: yScale(value) }
    })

    const timeTickCount = 4
    const timeTicks = Array.from({ length: timeTickCount + 1 }, (_, idx) => {
        const time = windowStart + (span / timeTickCount) * idx
        return { x: xScale(time), label: formatTime(new Date(time).toISOString()) }
    })

    return {
        width: resolvedWidth,
        height: chartLayout.height,
        padding: chartLayout.padding,
        ticks,
        timeTicks,
        series,
        startLabel: new Date(windowStart).toISOString(),
        endLabel: new Date(windowEnd).toISOString(),
    }
}
