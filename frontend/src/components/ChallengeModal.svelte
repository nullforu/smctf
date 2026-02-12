<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api, ApiError } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'
    import type { Challenge, Stack } from '../lib/types'
    import { getCategoryKey, getLocaleTag, localeStore, t } from '../lib/i18n'

    interface SubmissionState {
        status: 'idle' | 'loading' | 'success' | 'error'
        message?: string
    }

    interface Props {
        challenge: Challenge
        isSolved: boolean
        onClose: () => void
        onSolved: () => void
    }

    let { challenge, isSolved, onClose, onSolved }: Props = $props()

    let auth = $state(get(authStore))
    let flagInput = $state('')
    let submission = $state<SubmissionState>({ status: 'idle' })
    let isSuccessful = $derived(submission.status === 'success')
    let downloadLoading = $state(false)
    let downloadMessage = $state('')
    let stackInfo = $state<Stack | null>(null)
    let stackLoading = $state(false)
    let stackMessage = $state('')
    let stackPolling = $state(false)
    let stackNextInterval = $state(10000)

    const STACK_POLL_FAST_MS = 10000
    const STACK_POLL_SLOW_MS = 60000

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    const submitFlag = async () => {
        const translate = get(t)
        if (isSolved) {
            submission = { status: 'success', message: translate('challenge.correct') }
            return
        }

        if (submission.status === 'loading') return

        submission = { status: 'loading' }

        try {
            const result = await api.submitFlag(challenge.id, flagInput)

            if (result.correct) {
                submission = { status: 'success', message: translate('challenge.correct') }
                flagInput = ''
                onSolved()
            } else {
                submission = { status: 'error', message: translate('challenge.incorrect') }
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                submission = { status: 'success', message: translate('challenge.correct') }
                flagInput = ''
                onSolved()
                return
            }

            const formatted = formatApiError(error)
            submission = { status: 'error', message: formatted.message }
        }
    }

    const downloadFile = async () => {
        if (!challenge.has_file || downloadLoading) return

        downloadLoading = true
        downloadMessage = ''

        try {
            const result = await api.requestChallengeFileDownload(challenge.id)
            window.open(result.url, '_blank', 'noopener')
        } catch (error) {
            const formatted = formatApiError(error)
            downloadMessage = formatted.message
        } finally {
            downloadLoading = false
        }
    }

    const handleBackdropClick = (event: MouseEvent) => {
        if (event.target === event.currentTarget) {
            onClose()
        }
    }

    const formatTimestamp = (value?: string | null) => {
        if (!value) return get(t)('common.na')
        const date = new Date(value)
        if (Number.isNaN(date.getTime())) return value
        const locale = getLocaleTag(get(localeStore))
        return date.toLocaleString(locale)
    }

    const loadStack = async () => {
        if (!auth.user || !challenge.stack_enabled) return

        try {
            const result = await api.getStack(challenge.id)
            stackInfo = result
            stackNextInterval = stackInfo?.status === 'running' ? STACK_POLL_SLOW_MS : STACK_POLL_FAST_MS
            stackMessage = ''
        } catch (error) {
            if (error instanceof ApiError && error.status === 404) {
                stackInfo = null
                stackNextInterval = STACK_POLL_FAST_MS
                stackMessage = ''
                return
            }
            const formatted = formatApiError(error)
            stackMessage = formatted.message
        }
    }

    const createStack = async () => {
        const translate = get(t)
        if (isSolved) {
            stackMessage = translate('challenge.solvedCannotCreate')
            return
        }
        if (stackLoading || !auth.user) return
        stackLoading = true
        stackMessage = ''

        try {
            stackInfo = await api.createStack(challenge.id)
        } catch (error) {
            const formatted = formatApiError(error)
            stackMessage = formatted.message
        } finally {
            stackLoading = false
        }
    }

    const deleteStack = async () => {
        if (stackLoading || !auth.user) return
        stackLoading = true
        stackMessage = ''

        try {
            await api.deleteStack(challenge.id)
            stackInfo = null
        } catch (error) {
            const formatted = formatApiError(error)
            stackMessage = formatted.message
        } finally {
            stackLoading = false
        }
    }

    $effect(() => {
        if (!auth.user || !challenge.stack_enabled) {
            stackInfo = null
            stackMessage = ''
            stackPolling = false
            stackNextInterval = STACK_POLL_FAST_MS
            return
        }

        if (isSolved) {
            stackMessage = get(t)('challenge.stackSolvedNoNew')
        }

        loadStack()
    })

    $effect(() => {
        if (!auth.user || !challenge.stack_enabled || !stackInfo) {
            stackPolling = false
            return
        }

        stackPolling = true
        let timeoutId: ReturnType<typeof setTimeout>

        const poll = async () => {
            await loadStack()
            timeoutId = setTimeout(poll, stackNextInterval)
        }

        timeoutId = setTimeout(poll, stackNextInterval)
        return () => {
            clearTimeout(timeoutId)
            stackPolling = false
        }
    })
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-overlay/50 p-4" onclick={handleBackdropClick}>
    <div class="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl border border-border bg-surface p-8">
        <button
            class="absolute right-2 top-2 text-text-subtle hover:text-text"
            onclick={onClose}
            aria-label={$t('challenge.closeModal')}
        >
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
        </button>

        <div class="flex items-start justify-between gap-4">
            <div>
                <h2 class="text-2xl text-text">{challenge.title}</h2>
                <div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
                    <span class="rounded-full bg-surface-subtle px-3 py-1 text-xs font-medium text-text"
                        >{$t(getCategoryKey(challenge.category))}</span
                    >
                    <span class="text-text-muted">{$t('common.pointsShort', { points: challenge.points })}</span>
                    <span class="text-text-muted">
                        {$t('challenge.solvedCount', { count: challenge.solve_count })}
                    </span>
                </div>
            </div>
            {#if isSolved}
                <span class="rounded-full bg-success/20 px-4 py-1.5 text-sm text-success">
                    {$t('challenge.solvedLabel')}
                </span>
            {:else if !challenge.is_active}
                <span class="rounded-full bg-surface/10 px-4 py-1.5 text-sm text-text-muted">
                    {$t('challenge.inactiveLabel')}
                </span>
            {/if}
        </div>

        <div class="mt-6 text-text">
            <p class="whitespace-pre-wrap">{challenge.description}</p>
        </div>

        {#if challenge.has_file}
            <div class="mt-6">
                <div class="rounded-xl border border-border bg-surface-muted p-4 text-sm text-text">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <p class="font-medium">{$t('challenge.fileTitle')}</p>
                            <p class="text-xs text-text-subtle">
                                {challenge.file_name ?? 'challenge.zip'}
                            </p>
                        </div>
                        {#if auth.user}
                            <button
                                class="rounded-lg bg-contrast px-4 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60"
                                type="button"
                                onclick={downloadFile}
                                disabled={downloadLoading}
                            >
                                {downloadLoading ? $t('challenge.downloadPreparing') : $t('challenge.download')}
                            </button>
                        {/if}
                    </div>
                    {#if !auth.user}
                        <p class="mt-2 text-xs text-warning">{$t('challenge.fileLoginRequired')}</p>
                    {/if}
                    {#if downloadMessage}
                        <p class="mt-2 text-xs text-danger">{downloadMessage}</p>
                    {/if}
                </div>
            </div>
        {/if}

        <div class="mt-8 space-y-6">
            {#if challenge.stack_enabled}
                <div class="rounded-xl border border-border bg-surface-muted p-4 text-sm text-text">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <p class="font-medium">{$t('challenge.stackInstance')}</p>
                            <p class="text-xs text-text-subtle">
                                {stackPolling
                                    ? stackNextInterval === 60000
                                        ? $t('challenge.stackRefreshing60')
                                        : $t('challenge.stackRefreshing10')
                                    : $t('challenge.stackRefreshPaused')}
                            </p>
                        </div>
                        {#if auth.user}
                            <div class="flex flex-wrap items-center gap-2">
                                {#if stackInfo}
                                    <button
                                        class="rounded-lg border border-border px-3 py-2 text-xs font-medium text-text transition hover:border-border hover:text-text disabled:opacity-60"
                                        type="button"
                                        onclick={loadStack}
                                        disabled={stackLoading}
                                    >
                                        {stackLoading ? $t('common.loading') : $t('common.refresh')}
                                    </button>
                                    <button
                                        class="rounded-lg border border-danger/30 px-3 py-2 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60"
                                        type="button"
                                        onclick={deleteStack}
                                        disabled={stackLoading}
                                    >
                                        {stackLoading ? $t('challenge.stackWorking') : $t('challenge.deleteStack')}
                                    </button>
                                {:else}
                                    <button
                                        class="rounded-lg bg-contrast px-3 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60"
                                        type="button"
                                        onclick={createStack}
                                        disabled={stackLoading || isSolved}
                                    >
                                        {stackLoading ? $t('auth.creating') : $t('challenge.createStack')}
                                    </button>
                                {/if}
                            </div>
                        {/if}
                    </div>

                    {#if !auth.user}
                        <p class="mt-2 text-xs text-warning">{$t('challenge.stackLoginRequired')}</p>
                    {:else if isSolved}
                        <p class="mt-2 text-xs text-text-subtle">
                            {$t('challenge.stackSolvedNoNew')}
                        </p>
                    {:else if stackInfo}
                        <div class="mt-3 grid gap-2 text-xs text-text-muted">
                            <div class="flex flex-wrap items-center gap-2">
                                <span class="font-medium text-text">{$t('challenge.stackStatus')}</span>
                                <span class="rounded-full bg-surface-subtle px-2 py-0.5 text-[11px]">
                                    {stackInfo.status}
                                </span>
                            </div>
                            <div>
                                <span class="font-medium text-text">{$t('challenge.stackEndpoint')}</span>
                                <span class="ml-2">
                                    {stackInfo.node_public_ip && stackInfo.node_port
                                        ? `${stackInfo.node_public_ip}:${stackInfo.node_port}`
                                        : $t('challenge.stackPending')}
                                </span>
                            </div>
                            <div>
                                <span class="font-medium text-text">{$t('challenge.stackTtl')}</span>
                                <span class="ml-2">{formatTimestamp(stackInfo.ttl_expires_at)}</span>
                            </div>
                        </div>
                    {:else}
                        <p class="mt-2 text-xs text-text-subtle">
                            {$t('challenge.stackNoActive')}
                        </p>
                    {/if}

                    {#if stackMessage}
                        <p class="mt-2 text-xs text-danger">{stackMessage}</p>
                    {/if}
                </div>
            {/if}
            {#if !auth.user}
                <div class="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning-strong">
                    {$t('challenge.loginToSubmitPrefix')}
                    <a class="underline" href="/login" onclick={(e) => navigate('/login', e)}>{$t('auth.loginLink')}</a>
                    {$t('challenge.loginToSubmitSuffix')}
                </div>
            {:else if isSolved}
                <div class="rounded-xl border border-success/40 bg-success/10 p-4 text-sm text-success">
                    {$t('challenge.correct')}
                </div>
            {:else if !challenge.is_active}
                <div class="rounded-xl border border-border/40 bg-surface/10 p-4 text-sm text-text-muted">
                    {$t('challenge.inactiveMessage')}
                </div>
            {:else}
                <form
                    class="space-y-4"
                    onsubmit={(event) => {
                        event.preventDefault()
                        submitFlag()
                    }}
                >
                    <div>
                        <label class="block text-sm font-medium text-text mb-2"
                            >{$t('challenge.enterFlag')}
                            <input
                                class="w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                type="text"
                                bind:value={flagInput}
                                placeholder={$t('challenge.flagPlaceholder')}
                                autocomplete="off"
                            />
                        </label>
                    </div>
                    <button
                        class="w-full rounded-xl bg-accent py-3 text-sm font-medium text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                        type="submit"
                        disabled={submission.status === 'loading'}
                    >
                        {submission.status === 'loading' ? $t('challenge.submitting') : $t('challenge.submit')}
                    </button>
                    {#if submission.message}
                        <div
                            class={`rounded-xl border px-4 py-3 text-sm ${
                                isSuccessful
                                    ? 'border-success/40 bg-success/10 text-success '
                                    : 'border-danger/40 bg-danger/10 text-danger '
                            }`}
                        >
                            {submission.message}
                        </div>
                    {/if}
                </form>
            {/if}
        </div>
    </div>
</div>
