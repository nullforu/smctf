<script lang="ts">
    import { api } from '../lib/api'
    import { clearAuth, toggleTheme, themeStore, toggleThemeValue } from '../lib/stores'
    import { configStore } from '../lib/config'
    import { navigate } from '../lib/router'
    import type { AuthUser } from '../lib/types'
    import { get } from 'svelte/store'
    import { localeStore, setLocale, t, type Locale } from '../lib/i18n'

    const toggleThemeValueCallback = toggleThemeValue
    const toggleThemeCallback = toggleTheme
    const clearAuthCallback = clearAuth

    interface Props {
        user: AuthUser | null
    }

    let { user }: Props = $props()
    let theme = $state(get(themeStore))
    let appConfig = $state(get(configStore))
    let mobileMenuOpen = $state(false)
    let locale = $state<Locale>(get(localeStore))

    $effect(() => {
        const unsubscribe = themeStore.subscribe((value) => {
            theme = value
        })
        return unsubscribe
    })

    $effect(() => {
        const unsubscribe = configStore.subscribe((value) => {
            appConfig = value
        })
        return unsubscribe
    })

    $effect(() => {
        const unsubscribe = localeStore.subscribe((value) => {
            locale = value
        })
        return unsubscribe
    })

    function toggleMobileMenu() {
        mobileMenuOpen = !mobileMenuOpen
    }

    function closeMobileMenu() {
        mobileMenuOpen = false
    }

    function navigateAndClose(path: string, event: Event) {
        navigate(path, event)
        closeMobileMenu()
    }

    const handleLocaleChange = (event: Event) => {
        const target = event.currentTarget as HTMLSelectElement
        setLocale(target.value as Locale)
    }

    const logout = async (after?: () => void) => {
        try {
            await api.logout()
        } catch {
            clearAuthCallback()
        }
        navigate('/login')
        after?.()
    }
</script>

<header class="border-b border-border bg-surface/70 backdrop-blur">
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <button
            class="flex items-center justify-center p-2 text-text lg:hidden"
            onclick={toggleMobileMenu}
            aria-label={$t('header.toggleMobileMenu')}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
            >
                {#if mobileMenuOpen}
                    <line x1="18" y1="6" x2="6" y2="18" />
                    <line x1="6" y1="6" x2="18" y2="18" />
                {:else}
                    <line x1="3" y1="12" x2="21" y2="12" />
                    <line x1="3" y1="6" x2="21" y2="6" />
                    <line x1="3" y1="18" x2="21" y2="18" />
                {/if}
            </svg>
        </button>

        <a href="/" class="hidden items-center gap-4 lg:flex" onclick={(event) => navigate('/', event)}>
            <img
                src={`/logo_${toggleThemeValueCallback(theme)}_cropped.svg`}
                alt={$t('header.logoAlt')}
                class="h-6 w-auto"
            />
            <div>
                <p class="font-display text-xl text-text">{appConfig.header_title}</p>
                <p class="text-xs text-text-muted">{appConfig.header_description}</p>
            </div>
        </a>

        <nav class="hidden items-center gap-6 text-sm text-text lg:flex">
            <a class="hover:text-accent" href="/challenges" onclick={(e) => navigate('/challenges', e)}>
                {$t('nav.challenges')}
            </a>
            <a class="hover:text-accent" href="/scoreboard" onclick={(e) => navigate('/scoreboard', e)}>
                {$t('nav.scoreboard')}
            </a>
            <a class="hover:text-accent" href="/teams" onclick={(e) => navigate('/teams', e)}>
                {$t('nav.teams')}
            </a>
            <a class="hover:text-accent" href="/users" onclick={(e) => navigate('/users', e)}>
                {$t('nav.users')}
            </a>
            <a class="hover:text-accent" href="/profile" onclick={(e) => navigate('/profile', e)}>
                {$t('nav.profile')}
            </a>
            {#if user?.role === 'admin'}
                <a class="hover:text-accent" href="/admin" onclick={(e) => navigate('/admin', e)}>
                    {$t('nav.admin')}
                </a>
            {/if}
        </nav>

        <div class="hidden items-center gap-3 lg:flex">
            {#if user}
                <button class="hidden text-right text-xs text-text-muted sm:block" onclick={() => navigate('/profile')}>
                    <p class="text-text">{user.username}</p>
                    <p>{user.email}</p>
                </button>
                <button
                    class="rounded-full border border-border px-4 py-2 text-xs text-text transition hover:border-accent hover:text-accent"
                    onclick={() => logout()}
                >
                    {$t('auth.logout')}
                </button>
            {:else}
                <a
                    href="/login"
                    class="rounded-full border border-border px-4 py-2 text-xs text-text transition hover:border-accent hover:text-accent"
                    onclick={(e) => navigate('/login', e)}>{$t('auth.login')}</a
                >
                <a
                    href="/register"
                    class="rounded-full bg-accent/20 px-4 py-2 text-xs text-accent-strong transition hover:bg-accent/30"
                    onclick={(e) => navigate('/register', e)}>{$t('auth.register')}</a
                >
            {/if}
            <button
                class="rounded-full border border-border p-2 text-text transition hover:border-accent hover:text-accent"
                onclick={toggleThemeCallback}
                aria-label={$t('header.toggleTheme')}
                title={theme === 'light' ? $t('header.switchToDark') : $t('header.switchToLight')}
            >
                {#if theme === 'light'}
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                    >
                        <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
                    </svg>
                {:else}
                    <svg
                        xmlns="http://www.w3.org/2000/svg"
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                    >
                        <circle cx="12" cy="12" r="4" />
                        <path d="M12 2v2" />
                        <path d="M12 20v2" />
                        <path d="m4.93 4.93 1.41 1.41" />
                        <path d="m17.66 17.66 1.41 1.41" />
                        <path d="M2 12h2" />
                        <path d="M20 12h2" />
                        <path d="m6.34 17.66-1.41 1.41" />
                        <path d="m19.07 4.93-1.41 1.41" />
                    </svg>
                {/if}
            </button>
            <div class="flex items-center gap-2 rounded-full border border-border px-3 py-1.5 text-xs text-text">
                <select
                    class="bg-transparent text-xs text-text focus:outline-none cursor-pointer"
                    bind:value={locale}
                    onchange={handleLocaleChange}
                    aria-label={$t('header.language')}
                >
                    <option value="en">{$t('header.languageEnglish')}</option>
                    <option value="ko">{$t('header.languageKorean')}</option>
                    <option value="jp">{$t('header.languageJapanese')}</option>
                </select>
            </div>
        </div>
    </div>
</header>

{#if mobileMenuOpen}
    <button
        class="fixed inset-0 z-40 bg-overlay/50 backdrop-blur-sm lg:hidden"
        onclick={closeMobileMenu}
        aria-label={$t('header.closeMenu')}
    ></button>
{/if}

<aside
    class="fixed left-0 top-0 z-50 h-full w-72 transform border-r border-border bg-surface shadow-xl transition-transform duration-300 lg:hidden {mobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full'}"
>
    <div class="flex h-full flex-col">
        <div class="flex items-center justify-between border-b border-border p-6">
            <div class="flex items-center gap-3">
                <img
                    src={`/logo_${toggleThemeValueCallback(theme)}_cropped.svg`}
                    alt={$t('header.logoAlt')}
                    class="h-4 w-auto"
                />
                <div>
                    <p class="font-display text-xl text-text">{appConfig.header_title}</p>
                    <p class="text-xs text-text-muted">{appConfig.header_description}</p>
                </div>
            </div>
            <button class="p-1 text-text" onclick={closeMobileMenu} aria-label={$t('header.closeMenu')}>
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="24"
                    height="24"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <line x1="18" y1="6" x2="6" y2="18" />
                    <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
            </button>
        </div>

        <div class="flex flex-1 flex-col overflow-y-auto p-6">
            {#if user}
                <div class="mb-6 rounded-lg border border-border bg-surface-muted p-4">
                    <p class="text-sm font-medium text-text">{user.username}</p>
                    <p class="text-xs text-text-muted">{user.email}</p>
                    {#if user.role === 'admin'}
                        <span class="mt-2 inline-block rounded-full bg-accent/20 px-2 py-0.5 text-xs text-accent-strong"
                            >{$t('common.admin')}</span
                        >
                    {/if}
                </div>
            {/if}

            <nav class="flex flex-col gap-2">
                <a
                    href="/challenges"
                    class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                    onclick={(e) => navigateAndClose('/challenges', e)}
                >
                    {$t('nav.challenges')}
                </a>
                <a
                    href="/scoreboard"
                    class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                    onclick={(e) => navigateAndClose('/scoreboard', e)}
                >
                    {$t('nav.scoreboard')}
                </a>
                <a
                    href="/users"
                    class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                    onclick={(e) => navigateAndClose('/users', e)}
                >
                    {$t('nav.users')}
                </a>
                <a
                    href="/teams"
                    class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                    onclick={(e) => navigateAndClose('/teams', e)}
                >
                    {$t('nav.teams')}
                </a>
                <a
                    href="/profile"
                    class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                    onclick={(e) => navigateAndClose('/profile', e)}
                >
                    {$t('nav.profile')}
                </a>
                {#if user?.role === 'admin'}
                    <a
                        href="/admin"
                        class="rounded-lg px-4 py-3 text-sm text-text transition hover:bg-accent/10 hover:text-accent"
                        onclick={(e) => navigateAndClose('/admin', e)}
                    >
                        {$t('nav.admin')}
                    </a>
                {/if}
            </nav>

            <div class="my-6 border-t border-border"></div>

            <div class="flex flex-col gap-3">
                <div class="rounded-lg border border-border px-4 py-3 text-sm text-text">
                    <div class="flex items-center justify-between gap-3">
                        <span class="text-text-muted">{$t('header.language')}</span>
                        <select
                            class="bg-transparent text-sm text-text focus:outline-none"
                            bind:value={locale}
                            onchange={handleLocaleChange}
                            aria-label={$t('header.language')}
                        >
                            <option value="en">{$t('header.languageEnglish')}</option>
                            <option value="ko">{$t('header.languageKorean')}</option>
                            <option value="jp">{$t('header.languageJapanese')}</option>
                        </select>
                    </div>
                </div>
                <button
                    class="flex items-center justify-between rounded-lg border border-border px-4 py-3 text-sm text-text transition hover:border-accent hover:text-accent"
                    onclick={toggleThemeCallback}
                >
                    <span>{theme === 'light' ? $t('header.switchToDark') : $t('header.switchToLight')}</span>
                    {#if theme === 'light'}
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            width="18"
                            height="18"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
                        </svg>
                    {:else}
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            width="18"
                            height="18"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                        >
                            <circle cx="12" cy="12" r="4" />
                            <path d="M12 2v2" />
                            <path d="M12 20v2" />
                            <path d="m4.93 4.93 1.41 1.41" />
                            <path d="m17.66 17.66 1.41 1.41" />
                            <path d="M2 12h2" />
                            <path d="M20 12h2" />
                            <path d="m6.34 17.66-1.41 1.41" />
                            <path d="m19.07 4.93-1.41 1.41" />
                        </svg>
                    {/if}
                </button>

                {#if user}
                    <button
                        class="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-danger transition hover:border-danger/50 hover:bg-danger/20"
                        onclick={() => logout(closeMobileMenu)}
                    >
                        {$t('auth.logout')}
                    </button>
                {:else}
                    <a
                        href="/login"
                        class="rounded-lg border border-border px-4 py-3 text-center text-sm text-text transition hover:border-accent hover:text-accent"
                        onclick={(e) => navigateAndClose('/login', e)}
                    >
                        {$t('auth.login')}
                    </a>
                    <a
                        href="/register"
                        class="rounded-lg bg-accent/20 px-4 py-3 text-center text-sm text-accent-strong transition hover:bg-accent/30"
                        onclick={(e) => navigateAndClose('/register', e)}
                    >
                        {$t('auth.register')}
                    </a>
                {/if}
            </div>
        </div>
    </div>
</aside>
