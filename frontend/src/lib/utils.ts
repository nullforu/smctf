import { get } from 'svelte/store'
import type { ApiErrorDetail } from './api'
import { ApiError } from './api'
import { getLocaleTag, localeStore, t } from './i18n'

export type FieldErrors = Record<string, string>

export const formatApiError = (error: unknown) => {
    const translate = get(t)
    if (error instanceof ApiError) {
        const fieldErrors = buildFieldErrors(error.details)

        if (error.status === 429) {
            const resetSeconds = error.rateLimit?.reset_seconds
            const message =
                typeof resetSeconds === 'number'
                    ? translate('errors.tooManyRequests', { seconds: resetSeconds })
                    : translate('errors.tooManyRequestsLater')

            return { message, fieldErrors }
        }

        return { message: error.message, fieldErrors }
    }
    return { message: translate('errors.network'), fieldErrors: {} }
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

    const locale = getLocaleTag(get(localeStore))
    return date.toLocaleString(locale, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        timeZone: 'UTC',
    })
}

export const isZipFile = (file: File) => file.name.toLowerCase().endsWith('.zip')

export const parseRouteId = (value?: string) => {
    if (!value) return null
    const parsed = Number.parseInt(value, 10)
    return Number.isNaN(parsed) ? null : parsed
}
