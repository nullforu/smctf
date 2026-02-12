<script lang="ts">
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate } from '../lib/router'
    import FormMessage from '../components/FormMessage.svelte'

    let email = $state('')
    let username = $state('')
    let password = $state('')
    let registrationKey = $state('')
    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let success = $state(false)

    const submit = async () => {
        loading = true
        success = false
        errorMessage = ''
        fieldErrors = {}

        try {
            await api.register({ email, username, password, registration_key: registrationKey })

            success = true
            email = ''
            username = ''
            password = ''
            registrationKey = ''
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
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Register</h2>

            <form
                class="mt-6 space-y-5"
                onsubmit={(event) => {
                    event.preventDefault()
                    submit()
                }}
            >
                <div>
                    <label
                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                        for="register-email">Email</label
                    >
                    <input
                        id="register-email"
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
                        for="register-username">Username</label
                    >
                    <input
                        id="register-username"
                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                        type="text"
                        bind:value={username}
                        placeholder="user1"
                        autocomplete="username"
                    />
                    {#if fieldErrors.username}
                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">username: {fieldErrors.username}</p>
                    {/if}
                </div>
                <div>
                    <label
                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                        for="register-password">Password</label
                    >
                    <input
                        id="register-password"
                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                        type="password"
                        bind:value={password}
                        placeholder="••••••••"
                        autocomplete="new-password"
                    />
                    {#if fieldErrors.password}
                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">password: {fieldErrors.password}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="register-key"
                        >Registration Key</label
                    >
                    <input
                        id="register-key"
                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                        type="text"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        maxlength="6"
                        bind:value={registrationKey}
                        placeholder="6-digit key"
                        autocomplete="one-time-code"
                    />
                    {#if fieldErrors.registration_key}
                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                            registration_key: {fieldErrors.registration_key}
                        </p>
                    {/if}
                </div>

                {#if errorMessage}
                    <FormMessage variant="error" message={errorMessage} />
                {/if}
                {#if success}
                    <FormMessage variant="success">
                        Account created successfully. Please <a
                            class="underline"
                            href="/login"
                            onclick={(e) => navigate('/login', e)}>login</a
                        >.
                    </FormMessage>
                {/if}

                <button
                    class="w-full rounded-xl bg-teal-600 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? 'Creating...' : 'Create Account'}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-slate-200 bg-white p-10 dark:border-slate-800/80 dark:bg-slate-900/40">
            <h3 class="text-lg text-slate-900 dark:text-slate-100">Notice</h3>
            <ul class="mt-4 space-y-3 text-sm text-slate-700 dark:text-slate-400">
                <li>Please read and follow the competition rules.</li>
                <li>Participants may be restricted or disqualified if rules are violated.</li>
            </ul>
        </div>
    </div>
</section>
