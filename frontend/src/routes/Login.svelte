<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate as _navigate } from '../lib/router'

    const navigate = _navigate

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let email = $state('')
    let password = $state('')
    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let auth = $state(get(authStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

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
        <div class="rounded-3xl border border-slate-200 bg-white p-10 dark:border-slate-800/80 dark:bg-slate-900/40">
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Login</h2>

            {#if auth.user}
                <div
                    class="mt-6 rounded-xl border border-teal-500/40 bg-teal-500/10 p-4 text-sm text-teal-700 dark:text-teal-200"
                >
                    Already logged in as {auth.user.username}.
                    <a class="ml-2 underline" href="/challenges" onclick={(e) => navigate('/challenges', e)}
                        >Go to Challenges</a
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
                    <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="login-email"
                        >Email</label
                    >
                    <input
                        id="login-email"
                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                        type="email"
                        bind:value={email}
                        placeholder="user@example.com"
                        autocomplete="email"
                    />
                    {#if fieldErrors.email}
                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">email: {fieldErrors.email}</p>
                    {/if}
                </div>
                <div>
                    <label
                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                        for="login-password">Password</label
                    >
                    <input
                        id="login-password"
                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        autocomplete="current-password"
                    />
                    {#if fieldErrors.password}
                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">password: {fieldErrors.password}</p>
                    {/if}
                </div>

                {#if errorMessage}
                    <p
                        class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
                    >
                        {errorMessage}
                    </p>
                {/if}

                <button
                    class="w-full rounded-xl bg-teal-600 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? 'Logging in...' : 'Login'}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-slate-200 bg-white p-10 dark:border-slate-800/80 dark:bg-slate-900/40">
            <h3 class="text-lg text-slate-900 dark:text-slate-100">Need Help?</h3>
            <ul class="mt-4 space-y-3 text-sm text-slate-700 dark:text-slate-400">
                <li>
                    Don't have an account? <a
                        class="text-teal-600 underline dark:text-teal-200"
                        href="/register"
                        onclick={(e) => navigate('/register', e)}>Sign up</a
                    >.
                </li>
            </ul>
        </div>
    </div>
</section>
