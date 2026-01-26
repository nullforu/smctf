import type { TimelineSubmission, TimelineResponse } from '../lib/types'

export interface ChartPoint {
    x: number
    y: number
    value: number
}

export interface ChartSubmissionPoint {
    x: number
    y: number
    value: number
    submission: TimelineSubmission
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
    submissionPoints: ChartSubmissionPoint[]
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

    return date.toLocaleTimeString('ko-KR', {
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'Asia/Seoul',
    })
}

export const buildChartModel = (
    data: TimelineResponse,
    windowMinutesValue: number,
    widthValue: number,
): ChartModel | null => {
    const baseWidth = Math.floor(widthValue || chartLayout.width)
    const resolvedWidth = Math.max(chartLayout.width, baseWidth)

    if (!data.submissions || data.submissions.length === 0) return null

    const submissions = data.submissions
        .map((sub) => ({ sub, time: new Date(sub.timestamp).getTime() }))
        .filter((entry) => !Number.isNaN(entry.time))
        .sort((a, b) => a.time - b.time)

    if (submissions.length === 0) return null

    const times = submissions.map((entry) => entry.time)
    const windowStart = Math.min(...times)
    const windowEnd = Math.max(...times)
    const span = Math.max(1, windowEnd - windowStart)

    const plotWidth = resolvedWidth - chartLayout.padding.left - chartLayout.padding.right
    const plotHeight = chartLayout.height - chartLayout.padding.top - chartLayout.padding.bottom

    const userMap = new Map<number, string>()
    for (const entry of submissions) {
        if (!userMap.has(entry.sub.user_id)) {
            userMap.set(entry.sub.user_id, entry.sub.username)
        }
    }

    const userScores = new Map<number, number>()
    for (const entry of submissions) {
        const current = userScores.get(entry.sub.user_id) || 0
        userScores.set(entry.sub.user_id, current + entry.sub.points)
    }

    const topUsers = Array.from(userScores.entries())
        .sort((a, b) => b[1] - a[1])
        .slice(0, chartUserLimit)
        .map(([user_id]) => ({ user_id, username: userMap.get(user_id) || '' }))

    if (topUsers.length === 0) return null

    const submissionsByUser = new Map<number, TimelineSubmission[]>()

    for (const user of topUsers) {
        submissionsByUser.set(user.user_id, [])
    }

    for (const entry of submissions) {
        if (submissionsByUser.has(entry.sub.user_id)) {
            submissionsByUser.get(entry.sub.user_id)?.push(entry.sub)
        }
    }

    let maxValue = 0
    for (const userSubs of submissionsByUser.values()) {
        const total = userSubs.reduce((sum, sub) => sum + sub.points, 0)
        if (total > maxValue) maxValue = total
    }
    const safeMax = Math.max(1, maxValue)

    const xScale = (time: number) => chartLayout.padding.left + ((time - windowStart) / span) * plotWidth
    const yScale = (value: number) => chartLayout.padding.top + plotHeight - (value / safeMax) * plotHeight

    const series = topUsers.map((user, index) => {
        const userSubs = submissionsByUser.get(user.user_id) || []
        const submissionPoints: ChartSubmissionPoint[] = []
        let runningScore = 0

        for (const sub of userSubs) {
            const time = new Date(sub.timestamp).getTime()
            const clampedTime = Math.min(windowEnd, Math.max(windowStart, time))
            runningScore += sub.points
            submissionPoints.push({
                submission: sub,
                value: runningScore,
                x: xScale(clampedTime),
                y: yScale(runningScore),
            })
        }

        const points: ChartPoint[] = [
            { x: xScale(windowStart), y: yScale(0), value: 0 },
            ...submissionPoints.map((point) => ({ x: point.x, y: point.y, value: point.value })),
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
            submissionPoints,
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
