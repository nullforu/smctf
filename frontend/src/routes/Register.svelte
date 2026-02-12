<script lang="ts">
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'
    import { navigate } from '../lib/router'
    import FormMessage from '../components/FormMessage.svelte'
    import { t } from '../lib/i18n'

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
        <div class="rounded-3xl border border-border bg-surface p-10">
            <h2 class="text-3xl text-text">{$t('auth.register')}</h2>

            <form
                class="mt-6 space-y-5"
                onsubmit={(event) => {
                    event.preventDefault()
                    submit()
                }}
            >
                <div>
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="register-email"
                        >{$t('auth.emailLabel')}</label
                    >
                    <input
                        id="register-email"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="email"
                        bind:value={email}
                        placeholder={$t('auth.emailPlaceholder')}
                        autocomplete="email"
                    />
                    {#if fieldErrors.email}
                        <p class="mt-2 text-xs text-danger">{$t('auth.emailLabel')}: {fieldErrors.email}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="register-username"
                        >{$t('auth.usernameLabel')}</label
                    >
                    <input
                        id="register-username"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="text"
                        bind:value={username}
                        placeholder={$t('auth.usernamePlaceholder')}
                        autocomplete="username"
                    />
                    {#if fieldErrors.username}
                        <p class="mt-2 text-xs text-danger">{$t('auth.usernameLabel')}: {fieldErrors.username}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="register-password"
                        >{$t('auth.passwordLabel')}</label
                    >
                    <input
                        id="register-password"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="password"
                        bind:value={password}
                        placeholder={$t('auth.passwordPlaceholder')}
                        autocomplete="new-password"
                    />
                    {#if fieldErrors.password}
                        <p class="mt-2 text-xs text-danger">{$t('auth.passwordLabel')}: {fieldErrors.password}</p>
                    {/if}
                </div>
                <div>
                    <label class="text-xs uppercase tracking-wide text-text-muted" for="register-key"
                        >{$t('auth.registrationKey')}</label
                    >
                    <input
                        id="register-key"
                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                        type="text"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        maxlength="6"
                        bind:value={registrationKey}
                        placeholder={$t('auth.registrationKeyPlaceholder')}
                        autocomplete="one-time-code"
                    />
                    {#if fieldErrors.registration_key}
                        <p class="mt-2 text-xs text-danger">
                            {$t('auth.registrationKey')}: {fieldErrors.registration_key}
                        </p>
                    {/if}
                </div>

                {#if errorMessage}
                    <FormMessage variant="error" message={errorMessage} />
                {/if}
                {#if success}
                    <FormMessage variant="success">
                        {$t('auth.accountCreatedPrefix')}
                        <a class="underline" href="/login" onclick={(e) => navigate('/login', e)}
                            >{$t('auth.loginLink')}</a
                        >
                        {$t('auth.accountCreatedSuffix')}
                    </FormMessage>
                {/if}

                <button
                    class="w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                    type="submit"
                    disabled={loading}
                >
                    {loading ? $t('auth.creating') : $t('auth.createAccount')}
                </button>
            </form>
        </div>

        <div class="rounded-3xl border border-border bg-surface p-10">
            <h3 class="text-lg text-text">{$t('register.noticeTitle')}</h3>
            <ul class="mt-4 space-y-3 text-sm text-text">
                <li>{$t('register.noticeRule1')}</li>
                <li>{$t('register.noticeRule2')}</li>
            </ul>
        </div>
    </div>
</section>
