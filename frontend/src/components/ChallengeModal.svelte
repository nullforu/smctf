<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api, ApiError } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'
    import type { Challenge } from '../lib/types'

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

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    const submitFlag = async () => {
        if (isSolved) {
            submission = { status: 'success', message: '이미 해결한 문제입니다.' }
            return
        }

        if (submission.status === 'loading') return

        submission = { status: 'loading' }

        try {
            const result = await api.submitFlag(challenge.id, flagInput)

            if (result.correct) {
                submission = { status: 'success', message: '정답입니다!' }
                flagInput = ''
                onSolved()
            } else {
                submission = { status: 'error', message: '오답입니다. 다시 시도해 주세요.' }
            }
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                submission = { status: 'success', message: '정답입니다! (이미 해결됨)' }
                flagInput = ''
                onSolved()
                return
            }

            const formatted = formatApiError(error)
            submission = { status: 'error', message: formatted.message }
        }
    }

    const handleBackdropClick = (event: MouseEvent) => {
        if (event.target === event.currentTarget) {
            onClose()
        }
    }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onclick={handleBackdropClick}>
    <div
        class="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl border border-slate-200 bg-white p-8 dark:border-slate-800/80 dark:bg-slate-900/95"
    >
        <button
            class="absolute right-2 top-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
            onclick={onClose}
            aria-label="Close Modal"
        >
            <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
        </button>

        <div class="flex items-start justify-between gap-4">
            <div>
                <h2 class="text-2xl text-slate-900 dark:text-slate-100">{challenge.title}</h2>
                <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">{challenge.points} pts</p>
            </div>
            {#if isSolved}
                <span class="rounded-full bg-emerald-500/20 px-4 py-1.5 text-sm text-emerald-700 dark:text-emerald-200"
                    >Solved</span
                >
            {:else if !challenge.is_active}
                <span
                    class="rounded-full bg-slate-400/10 px-4 py-1.5 text-sm text-slate-600 dark:bg-slate-500/10 dark:text-slate-300"
                    >Inactive</span
                >
            {/if}
        </div>

        <div class="mt-6 text-slate-700 dark:text-slate-300">
            <p class="whitespace-pre-wrap">{challenge.description}</p>
        </div>

        <div class="mt-8">
            {#if !auth.user}
                <div
                    class="rounded-xl border border-amber-500/40 bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-100"
                >
                    플래그 제출은 로그인 후 가능합니다.
                    <a class="ml-1 underline" href="/login" onclick={(e) => navigate('/login', e)}>로그인</a>
                </div>
            {:else if isSolved}
                <div
                    class="rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-sm text-emerald-700 dark:text-emerald-200"
                >
                    이미 해결한 문제입니다.
                </div>
            {:else if !challenge.is_active}
                <div
                    class="rounded-xl border border-slate-400/40 bg-slate-400/10 p-4 text-sm text-slate-600 dark:text-slate-400"
                >
                    비활성화된 문제입니다.
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
                        <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2"
                            >플래그 입력
                            <input
                                class="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                type="text"
                                bind:value={flagInput}
                                placeholder="flag&#123;...&#125;"
                                autocomplete="off"
                            />
                        </label>
                    </div>
                    <button
                        class="w-full rounded-xl bg-teal-600 py-3 text-sm font-medium text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                        type="submit"
                        disabled={submission.status === 'loading'}
                    >
                        {submission.status === 'loading' ? '제출 중...' : '제출'}
                    </button>
                    {#if submission.message}
                        <div
                            class={`rounded-xl border px-4 py-3 text-sm ${
                                isSuccessful
                                    ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-200'
                                    : 'border-rose-500/40 bg-rose-500/10 text-rose-700 dark:text-rose-200'
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
