<script lang="ts">
    import { onDestroy } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, type AuthState } from '../lib/stores'
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'

    let title = $state('')
    let description = $state('')
    let points = $state(100)
    let flag = $state('')
    let isActive = $state(true)

    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let successMessage = $state('')
    let auth = $state<AuthState>(get(authStore))
    const unsubscribe = authStore.subscribe((value) => {
        auth = value
    })
    onDestroy(unsubscribe)

    const submit = async () => {
        loading = true
        errorMessage = ''
        successMessage = ''
        fieldErrors = {}
        try {
            const created = await api.createChallenge({
                title,
                description,
                points: Number(points),
                flag,
                is_active: isActive,
            })
            successMessage = `문제 "${created.title}" (ID ${created.id}) 생성 완료`
            title = ''
            description = ''
            points = 100
            flag = ''
            isActive = true
        } catch (error) {
            const formatted = formatApiError(error)
            errorMessage = formatted.message
            fieldErrors = formatted.fieldErrors
        } finally {
            loading = false
        }
    }
</script>

<section class="fade-in">
    <div>
        <h2 class="text-3xl text-slate-100">Admin</h2>
        <p class="mt-2 text-sm text-slate-400">새로운 문제를 생성하고 활성화합니다.</p>
    </div>

    {#if !auth.user}
        <div class="mt-6 rounded-2xl border border-amber-500/40 bg-amber-500/10 p-6 text-sm text-amber-100">
            관리자 기능은 로그인 후 접근 가능합니다.
        </div>
    {:else if auth.user.role !== 'admin'}
        <div class="mt-6 rounded-2xl border border-rose-500/40 bg-rose-500/10 p-6 text-sm text-rose-200">
            권한이 없습니다. 관리자 계정으로 로그인하세요.
        </div>
    {:else}
        <div class="mt-6 rounded-3xl border border-slate-800/80 bg-slate-900/40 p-8">
            <form
                class="space-y-5"
                onsubmit={(event) => {
                    event.preventDefault()
                    submit()
                }}
            >
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="admin-title">Title</label>
                    <input
                        id="admin-title"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        type="text"
                        bind:value={title}
                    />
                    {#if fieldErrors.title}
                        <p class="mt-2 text-xs text-rose-300">title: {fieldErrors.title}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="admin-description"
                        >Description</label
                    >
                    <textarea
                        id="admin-description"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        rows="5"
                        bind:value={description}
                    ></textarea>
                    {#if fieldErrors.description}
                        <p class="mt-2 text-xs text-rose-300">description: {fieldErrors.description}</p>
                    {/if}
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                    <div>
                        <label class="text-xs uppercase tracking-wide text-slate-400" for="admin-points">Points</label>
                        <input
                            id="admin-points"
                            class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                            type="number"
                            min="1"
                            bind:value={points}
                        />
                        {#if fieldErrors.points}
                            <p class="mt-2 text-xs text-rose-300">points: {fieldErrors.points}</p>
                        {/if}
                    </div>
                    <div>
                        <label class="text-xs uppercase tracking-wide text-slate-400" for="admin-flag">Flag</label>
                        <input
                            id="admin-flag"
                            class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                            type="text"
                            bind:value={flag}
                        />
                        {#if fieldErrors.flag}
                            <p class="mt-2 text-xs text-rose-300">flag: {fieldErrors.flag}</p>
                        {/if}
                    </div>
                </div>
                <label class="flex items-center gap-3 text-sm text-slate-300">
                    <input type="checkbox" bind:checked={isActive} class="h-4 w-4 rounded border-slate-700" />
                    활성화 상태로 생성
                </label>

                {#if errorMessage}
                    <p class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-200">
                        {errorMessage}
                    </p>
                {/if}
                {#if successMessage}
                    <p class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-200">
                        {successMessage}
                    </p>
                {/if}

                <button
                    class="w-full rounded-xl bg-teal-500/30 py-3 text-sm text-teal-100 transition hover:bg-teal-500/40 disabled:opacity-60"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? '생성 중...' : '문제 생성'}
                </button>
            </form>
        </div>
    {/if}
</section>
