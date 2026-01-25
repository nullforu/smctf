<script lang="ts">
    import { onDestroy, onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, type AuthState } from '../lib/stores'
    import { api, ApiError } from '../lib/api'
    import type { Challenge } from '../lib/types'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'

    interface SubmissionState {
        status: 'idle' | 'loading' | 'success' | 'error'
        message?: string
    }

    let challenges: Challenge[] = $state([])
    let loading = $state(true)
    let errorMessage = $state('')
    let solvedIds = $state(new Set<number>())
    let openId: number | null = $state(null)
    let flagInputs: Record<number, string> = $state({})
    let submissions: Record<number, SubmissionState> = $state({})
    let auth = $state<AuthState>(get(authStore))

    onDestroy(
        authStore.subscribe((value) => {
            auth = value
        }),
    )

    const onNav = (event: MouseEvent, path: string) => {
        event.preventDefault()
        navigate(path)
    }

    const loadChallenges = async () => {
        loading = true
        errorMessage = ''

        try {
            challenges = await api.challenges()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const loadSolved = async () => {
        if (!auth.user) return

        try {
            const solved = await api.solved()
            solvedIds = new Set(solved.map((item) => item.challenge_id))
        } catch {
            solvedIds = new Set()
        }
    }

    const setSubmission = (id: number, state: SubmissionState) => {
        submissions = { ...submissions, [id]: state }
    }

    const setFlagInput = (id: number, value: string) => {
        flagInputs = { ...flagInputs, [id]: value }
    }

    const submitFlag = async (id: number) => {
        if (solvedIds.has(id)) {
            setSubmission(id, { status: 'success', message: '이미 해결한 문제입니다.' })
            return
        }

        if (submissions[id]?.status === 'loading') return
        setSubmission(id, { status: 'loading' })

        const flag = flagInputs[id]

        try {
            const result = await api.submitFlag(id, flag)

            if (result.correct) {
                setSubmission(id, { status: 'success', message: '정답입니다!' })
                solvedIds = new Set([...solvedIds, id])
                setFlagInput(id, '')
            } else {
                setSubmission(id, { status: 'error', message: '오답입니다. 다시 시도해 주세요.' })
            }

            await loadSolved()
        } catch (error) {
            if (error instanceof ApiError && error.status === 409) {
                setSubmission(id, { status: 'success', message: '정답입니다! (이미 해결됨)' })
                solvedIds = new Set([...solvedIds, id])
                setFlagInput(id, '')

                return
            }

            const formatted = formatApiError(error)
            setSubmission(id, { status: 'error', message: formatted.message })
        }
    }

    onMount(async () => {
        await loadChallenges()
        await loadSolved()
    })
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-100">Challenges</h2>
        </div>
        <div class="rounded-full border border-slate-800/70 bg-slate-900/40 px-4 py-2 text-xs text-slate-300">
            푼 문제 {solvedIds.size} / 전체 {challenges.filter((c) => c.is_active).length}
            {challenges.filter((c) => !c.is_active).length > 0
                ? `(비활성 문제 ${challenges.filter((c) => !c.is_active).length}개)`
                : ''}
        </div>
    </div>

    {#if loading}
        <div class="mt-6 rounded-2xl border border-slate-800/70 bg-slate-900/40 p-8 text-center text-slate-400">
            문제를 불러오는 중...
        </div>
    {:else if errorMessage}
        <div class="mt-6 rounded-2xl border border-rose-500/40 bg-rose-500/10 p-6 text-sm text-rose-200">
            {errorMessage}
        </div>
    {:else}
        <div class="mt-6 grid gap-6 md:grid-cols-2">
            {#each challenges as challenge}
                <div
                    class={`rounded-2xl border p-6 transition ${
                        challenge.is_active
                            ? 'border-slate-800/80 bg-slate-900/40'
                            : 'border-slate-800/40 bg-slate-900/20 opacity-60'
                    }`}
                >
                    <div class="flex items-start justify-between">
                        <div>
                            <h3 class="text-lg text-slate-100">{challenge.title}</h3>
                            <p class="mt-1 text-xs text-slate-400">{challenge.points} pts</p>
                        </div>
                        {#if solvedIds.has(challenge.id)}
                            <span class="rounded-full bg-emerald-500/20 px-3 py-1 text-xs text-emerald-200">Solved</span
                            >
                        {:else if !challenge.is_active}
                            <span class="rounded-full bg-slate-500/10 px-3 py-1 text-xs text-slate-300">Inactive</span>
                        {/if}
                    </div>

                    <p class="mt-4 text-sm text-slate-300">{challenge.description}</p>

                    <div class="mt-6 flex flex-wrap items-center gap-3">
                        <button
                            class="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-200 transition hover:border-teal-400 disabled:cursor-not-allowed disabled:opacity-60"
                            onclick={() => (openId = openId === challenge.id ? null : challenge.id)}
                            disabled={!challenge.is_active || solvedIds.has(challenge.id)}
                        >
                            {solvedIds.has(challenge.id)
                                ? '해결 완료'
                                : openId === challenge.id
                                  ? '닫기'
                                  : '플래그 제출'}
                        </button>
                    </div>

                    {#if openId === challenge.id}
                        {#if !auth.user}
                            <div
                                class="mt-4 rounded-xl border border-amber-500/40 bg-amber-500/10 p-4 text-xs text-amber-100"
                            >
                                플래그 제출은 로그인 후 가능합니다.
                                <a class="ml-1 underline" href="/login" onclick={(event) => onNav(event, '/login')}
                                    >로그인</a
                                >
                            </div>
                        {:else if submissions[challenge.id]?.status === 'success'}
                            <div
                                class="mt-4 rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-xs text-emerald-200"
                            >
                                {submissions[challenge.id]?.message ?? '정답입니다!'}
                            </div>
                        {:else if solvedIds.has(challenge.id)}
                            <div
                                class="mt-4 rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-xs text-emerald-200"
                            >
                                이미 해결한 문제입니다.
                            </div>
                        {:else}
                            <form
                                class="mt-4 space-y-3"
                                onsubmit={(event) => {
                                    event.preventDefault()
                                    submitFlag(challenge.id)
                                }}
                            >
                                <input
                                    class="w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                                    type="text"
                                    bind:value={flagInputs[challenge.id]}
                                    placeholder="flag&#123;...&#125;"
                                    autocomplete="off"
                                />
                                <button
                                    class="w-full rounded-xl bg-teal-500/30 py-2 text-sm text-teal-100 transition hover:bg-teal-500/40 disabled:opacity-60"
                                    type="submit"
                                    disabled={submissions[challenge.id]?.status === 'loading'}
                                >
                                    {submissions[challenge.id]?.status === 'loading' ? '제출 중...' : '제출'}
                                </button>
                                {#if submissions[challenge.id]?.message}
                                    <p
                                        class={`rounded-xl border px-4 py-2 text-xs ${
                                            submissions[challenge.id]?.status === 'success'
                                                ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-200'
                                                : 'border-rose-500/40 bg-rose-500/10 text-rose-200'
                                        }`}
                                    >
                                        {submissions[challenge.id]?.message}
                                    </p>
                                {/if}
                            </form>
                        {/if}
                    {/if}
                </div>
            {/each}
        </div>
    {/if}
</section>
