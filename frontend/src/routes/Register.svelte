<script lang="ts">
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate } from '../lib/router'

    let email = $state('')
    let username = $state('')
    let password = $state('')
    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let success = $state(false)
    const onNav = (event: MouseEvent, path: string) => {
        event.preventDefault()
        navigate(path)
    }

    const submit = async () => {
        loading = true
        success = false
        errorMessage = ''
        fieldErrors = {}
        try {
            await api.register({ email, username, password })
            success = true
            email = ''
            username = ''
            password = ''
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
            <h2 class="text-3xl text-slate-100">계정 생성</h2>
            <p class="mt-2 text-sm text-slate-400">당신의 해킹 스토리를 시작하세요.</p>

            <form
                class="mt-6 space-y-5"
                onsubmit={(event) => {
                    event.preventDefault()
                    submit()
                }}
            >
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="register-email">Email</label>
                    <input
                        id="register-email"
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
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="register-username"
                        >Username</label
                    >
                    <input
                        id="register-username"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        type="text"
                        bind:value={username}
                        placeholder="user1"
                        autocomplete="username"
                    />
                    {#if fieldErrors.username}
                        <p class="mt-2 text-xs text-rose-300">username: {fieldErrors.username}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-400" for="register-password"
                        >Password</label
                    >
                    <input
                        id="register-password"
                        class="mt-2 w-full rounded-xl border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-100 focus:border-teal-400 focus:outline-none"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        autocomplete="new-password"
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
                {#if success}
                    <p class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-200">
                        계정이 생성되었습니다. 이제 <a
                            class="underline"
                            href="/login"
                            onclick={(event) => onNav(event, '/login')}>로그인</a
                        > 하세요.
                    </p>
                {/if}

                <button
                    class="w-full rounded-xl bg-teal-500/30 py-3 text-sm text-teal-100 transition hover:bg-teal-500/40 disabled:opacity-60"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? '생성 중...' : '계정 만들기'}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10">
            <h3 class="text-lg text-slate-100">가입 전 체크리스트</h3>
            <ul class="mt-4 space-y-3 text-sm text-slate-400">
                <li>이메일은 로그인과 알림에 사용됩니다.</li>
                <li>사용자명은 스코어보드에 표시됩니다.</li>
                <li>비밀번호는 최소 8자 이상을 권장합니다.</li>
            </ul>
        </div>
    </div>
</section>
