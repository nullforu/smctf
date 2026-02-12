<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate } from '../lib/router'
    import FormMessage from '../components/FormMessage.svelte'

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
        <div class="rounded-3xl border border-border bg-surface p-10">
            <h2 class="text-3xl text-text">Login</h2>

            {#if auth.user}
                <div class="mt-6 rounded-xl border border-accent/40 bg-accent/10 p-4 text-sm text-accent-strong">
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
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="login-email">Email</label>
                    <input
                        id="login-email"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="email"
                        bind:value={email}
                        placeholder="user@example.com"
                        autocomplete="email"
                    />
                    {#if fieldErrors.email}
                        <p class="mt-2 text-xs text-danger">email: {fieldErrors.email}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="login-password">Password</label>
                    <input
                        id="login-password"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        autocomplete="current-password"
                    />
                    {#if fieldErrors.password}
                        <p class="mt-2 text-xs text-danger">password: {fieldErrors.password}</p>
                    {/if}
                </div>

                {#if errorMessage}
                    <FormMessage variant="error" message={errorMessage} />
                {/if}

                <button
                    class="w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? 'Logging in...' : 'Login'}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-border bg-surface p-10">
            <h3 class="text-lg text-text">Need Help?</h3>
            <ul class="mt-4 space-y-3 text-sm text-text">
                <li>
                    Don't have an account? <a
                        class="text-accent underline"
                        href="/register"
                        onclick={(e) => navigate('/register', e)}>Sign up</a
                    >.
                </li>
            </ul>
        </div>
    </div>
</section>
