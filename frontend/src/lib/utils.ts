import type { ApiErrorDetail } from './api'
import { ApiError } from './api'

export type FieldErrors = Record<string, string>

export const formatApiError = (error: unknown) => {
    if (error instanceof ApiError) {
        const fieldErrors = buildFieldErrors(error.details)
        if (error.status === 429) {
            const resetSeconds = error.rateLimit?.reset_seconds
            const message =
                typeof resetSeconds === 'number'
                    ? `제출이 너무 많습니다. ${resetSeconds}초 후 다시 시도하세요.`
                    : '제출이 너무 많습니다. 잠시 후 다시 시도하세요.'
            return { message, fieldErrors }
        }
        return {
            message: error.message,
            fieldErrors,
        }
    }
    return { message: 'unexpected error', fieldErrors: {} }
}

const buildFieldErrors = (details?: ApiErrorDetail[]) => {
    if (!details || details.length === 0) return {} as FieldErrors
    return details.reduce<FieldErrors>((acc, detail) => {
        acc[detail.field] = detail.reason
        return acc
    }, {})
}

export const formatDateTime = (value: string) => {
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleString('ko-KR', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    })
}
