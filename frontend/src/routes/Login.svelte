<script lang="ts">
    import { onDestroy } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, type AuthState } from '../lib/stores'
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate } from '../lib/router'

    let email = $state('')
    let password = $state('')
    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let auth = $state<AuthState>(get(authStore))
    const unsubscribe = authStore.subscribe((value) => {
        auth = value
    })
    onDestroy(unsubscribe)
    const onNav = (event: MouseEvent, path: string) => {
        event.preventDefault()
        navigate(path)
    }

    const submit = async () => {
        loading = true
        errorMessage = ''
        fieldErrors = {}
        try {
            await api.login({ email, password })
            navigate('/challenges')
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
    <div class="grid gap-8 md:grid-cols-[1.1fr_1fr]">
        <div class="rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10">
            <h2 class="text-3xl text-slate-100">로그인</h2>
            <p class="mt-2 text-sm text-slate-400">세션을 이어서 문제를 해결하세요.</p>

            {#if auth.user}
                <div class="mt-6 rounded-xl border border-teal-500/40 bg-teal-500/10 p-4 text-sm text-teal-200">
                    이미 {auth.user.username} 계정으로 로그인되어 있습니다.
                    <a class="ml-2 underline" href="/challenges" onclick={(event) => onNav(event, '/challenges')}
                        >바로 이동</a
                    >
                </div>
            {/if}

            <form
                class="mt-6 space-y-5"
                onsubmit={(event) => {
                    event.preventDefault()
                    submit()
                }}
            >
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="login-email">Email</label>
                    <input
                        id="login-email"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        type="email"
                        bind:value={email}
                        placeholder="user@example.com"
                        autocomplete="email"
                    />
                    {#if fieldErrors.email}
                        <p class="mt-2 text-xs text-rose-300">email: {fieldErrors.email}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="login-password">Password</label>
                    <input
                        id="login-password"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        autocomplete="current-password"
                    />
                    {#if fieldErrors.password}
                        <p class="mt-2 text-xs text-rose-300">password: {fieldErrors.password}</p>
                    {/if}
                </div>

                {#if errorMessage}
                    <p class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-200">
                        {errorMessage}
                    </p>
                {/if}

                <button
                    class="w-full rounded-xl bg-teal-500/30 py-3 text-sm text-teal-100 transition hover:bg-teal-500/40 disabled:opacity-60"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? '로그인 중...' : '로그인'}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10">
            <h3 class="text-lg text-slate-100">도움이 필요하신가요?</h3>
            <ul class="mt-4 space-y-3 text-sm text-slate-400">
                <li>로그인이 실패한다면 이메일/비밀번호 입력을 다시 확인하세요.</li>
                <li>관리자가 계정을 승인한 경우 즉시 로그인 가능합니다.</li>
                <li>
                    아직 계정이 없다면 <a
                        class="text-teal-200 underline"
                        href="/register"
                        onclick={(event) => onNav(event, '/register')}>가입</a
                    >으로 이동하세요.
                </li>
            </ul>
        </div>
    </div>
</section>
