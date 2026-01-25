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
        onSolved: () => void
    }

    let { challenge, isSolved, onSolved }: Props = $props()

    let auth = $state(get(authStore))
    let openFlag = $state(false)
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
</script>

<div
    class={`rounded-2xl border p-6 transition ${
        challenge.is_active ? 'border-slate-800/80 bg-slate-900/40' : 'border-slate-800/40 bg-slate-900/20 opacity-60'
    }`}
>
    <div class="flex items-start justify-between">
        <div>
            <h3 class="text-lg text-slate-100">{challenge.title}</h3>
            <p class="mt-1 text-xs text-slate-400">{challenge.points} pts</p>
        </div>
        {#if isSolved}
            <span class="rounded-full bg-emerald-500/20 px-3 py-1 text-xs text-emerald-200">Solved</span>
        {:else if !challenge.is_active}
            <span class="rounded-full bg-slate-500/10 px-3 py-1 text-xs text-slate-300">Inactive</span>
        {/if}
    </div>

    <p class="mt-4 text-sm text-slate-300">{challenge.description}</p>

    <div class="mt-6 flex flex-wrap items-center gap-3">
        <button
            class="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-200 transition hover:border-teal-400 disabled:cursor-not-allowed disabled:opacity-60"
            onclick={() => (openFlag = !openFlag)}
            disabled={!challenge.is_active || isSolved}
        >
            {isSolved ? '해결 완료' : openFlag ? '닫기' : '플래그 제출'}
        </button>
    </div>

    {#if openFlag}
        {#if !auth.user}
            <div class="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/10 p-4 text-xs text-amber-100">
                플래그 제출은 로그인 후 가능합니다.
                <a class="ml-1 underline" href="/login" onclick={(e) => navigate('/login', e)}>로그인</a>
            </div>
        {:else if submission.status === 'success'}
            <div class="mt-4 rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-xs text-emerald-200">
                {submission.message ?? '정답입니다!'}
            </div>
        {:else if isSolved}
            <div class="mt-4 rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-xs text-emerald-200">
                이미 해결한 문제입니다.
            </div>
        {:else}
            <form
                class="mt-4 space-y-3"
                onsubmit={(event) => {
                    event.preventDefault()
                    submitFlag()
                }}
            >
                <input
                    class="w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                    type="text"
                    bind:value={flagInput}
                    placeholder="flag&#123;...&#125;"
                    autocomplete="off"
                />
                <button
                    class="w-full rounded-xl bg-teal-500/30 py-2 text-sm text-teal-100 transition hover:bg-teal-500/40 disabled:opacity-60"
                    type="submit"
                    disabled={submission.status === 'loading'}
                >
                    {submission.status === 'loading' ? '제출 중...' : '제출'}
                </button>
                {#if submission.message}
                    <p
                        class={`rounded-xl border px-4 py-2 text-xs ${
                            isSuccessful
                                ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200'
                                : 'border-rose-500/40 bg-rose-500/10 text-rose-200'
                        }`}
                    >
                        {submission.message}
                    </p>
                {/if}
            </form>
        {/if}
    {/if}
</div>
