<script lang="ts">
    import { api } from '../lib/api'
    import { clearAuth, toggleTheme, themeStore } from '../lib/stores'
    import { navigate } from '../lib/router'
    import type { AuthUser } from '../lib/types'
    import { get } from 'svelte/store'

    const toggleThemeCallback = toggleTheme
    const clearAuthCallback = clearAuth

    interface Props {
        user: AuthUser | null
    }

    let { user }: Props = $props()
    let theme = $state(get(themeStore))
    let mobileMenuOpen = $state(false)

    $effect(() => {
        const unsubscribe = themeStore.subscribe((value) => {
            theme = value
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
        navigate(path, event as MouseEvent)
        closeMobileMenu()
    }
</script>

<header class="border-b border-slate-200 bg-white/70 backdrop-blur dark:border-slate-800/70 dark:bg-slate-950/70">
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <button
            class="flex items-center justify-center p-2 text-slate-700 dark:text-slate-200 md:hidden"
            onclick={toggleMobileMenu}
            aria-label="Toggle mobile menu"
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

        <a href="/" class="hidden items-center gap-3 md:flex" onclick={(event) => navigate('/', event)}>
            <div
                class="flex h-10 w-10 items-center justify-center rounded-lg bg-teal-500/10 text-teal-600 dark:text-teal-300"
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
                    aria-hidden="true"
                >
                    <g transform="rotate(15 12 12)">
                        <path d="M6 3v18" />
                        <path d="M6 4h10l-2 3 2 3H6" />
                    </g>
                </svg>
            </div>
            <div>
                <p class="font-display text-xl text-slate-900 dark:text-slate-100">CTF</p>
                <p class="text-xs text-slate-600 dark:text-slate-400">Capture The Flag</p>
            </div>
        </a>

        <nav class="hidden items-center gap-6 text-sm text-slate-700 dark:text-slate-300 md:flex">
            <a
                class="hover:text-teal-600 dark:hover:text-teal-200"
                href="/challenges"
                onclick={(e) => navigate('/challenges', e)}>Challenges</a
            >
            <a
                class="hover:text-teal-600 dark:hover:text-teal-200"
                href="/scoreboard"
                onclick={(e) => navigate('/scoreboard', e)}>Scoreboard</a
            >
            <a class="hover:text-teal-600 dark:hover:text-teal-200" href="/users" onclick={(e) => navigate('/users', e)}
                >Users</a
            >
            <a
                class="hover:text-teal-600 dark:hover:text-teal-200"
                href="/profile"
                onclick={(e) => navigate('/profile', e)}>Profile</a
            >
            {#if user?.role === 'admin'}
                <a
                    class="hover:text-teal-600 dark:hover:text-teal-200"
                    href="/admin"
                    onclick={(e) => navigate('/admin', e)}>Admin</a
                >
            {/if}
        </nav>

        <div class="hidden items-center gap-3 md:flex">
            {#if user}
                <button
                    class="hidden text-right text-xs text-slate-600 dark:text-slate-400 sm:block"
                    onclick={() => navigate('/profile')}
                >
                    <p class="text-slate-900 dark:text-slate-200">{user.username}</p>
                    <p>{user.email}</p>
                </button>
                <button
                    class="rounded-full border border-slate-300 px-4 py-2 text-xs text-slate-800 transition hover:border-teal-500 hover:text-teal-600 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 dark:hover:text-teal-200"
                    onclick={async () => {
                        try {
                            await api.logout()
                        } catch {
                            clearAuthCallback()
                        }
                        navigate('/login')
                    }}
                >
                    Logout
                </button>
            {:else}
                <a
                    href="/login"
                    class="rounded-full border border-slate-300 px-4 py-2 text-xs text-slate-800 transition hover:border-teal-500 hover:text-teal-600 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 dark:hover:text-teal-200"
                    onclick={(e) => navigate('/login', e)}>Login</a
                >
                <a
                    href="/register"
                    class="rounded-full bg-teal-500/20 px-4 py-2 text-xs text-teal-700 transition hover:bg-teal-500/30 dark:text-teal-200"
                    onclick={(e) => navigate('/register', e)}>Register</a
                >
            {/if}
            <button
                class="rounded-full border border-slate-300 p-2 text-slate-700 transition hover:border-teal-500 hover:text-teal-600 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 dark:hover:text-teal-200"
                onclick={toggleThemeCallback}
                aria-label="Toggle theme"
                title={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
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
        </div>
    </div>
</header>

{#if mobileMenuOpen}
    <button
        class="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm md:hidden"
        onclick={closeMobileMenu}
        aria-label="Close menu"
    ></button>
{/if}

<aside
    class="fixed left-0 top-0 z-50 h-full w-72 transform border-r border-slate-200 bg-white shadow-xl transition-transform duration-300 dark:border-slate-800 dark:bg-slate-950 md:hidden {mobileMenuOpen
        ? 'translate-x-0'
        : '-translate-x-full'}"
>
    <div class="flex h-full flex-col">
        <div class="flex items-center justify-between border-b border-slate-200 p-6 dark:border-slate-800">
            <div class="flex items-center gap-3">
                <div
                    class="flex h-10 w-10 items-center justify-center rounded-lg bg-teal-500/10 text-teal-600 dark:text-teal-300"
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
                        <g transform="rotate(15 12 12)">
                            <path d="M6 3v18" />
                            <path d="M6 4h10l-2 3 2 3H6" />
                        </g>
                    </svg>
                </div>
                <div>
                    <p class="font-display text-xl text-slate-900 dark:text-slate-100">SMCTF</p>
                </div>
            </div>
            <button class="p-1 text-slate-700 dark:text-slate-200" onclick={closeMobileMenu} aria-label="Close menu">
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
                <div
                    class="mb-6 rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-900"
                >
                    <p class="text-sm font-medium text-slate-900 dark:text-slate-200">{user.username}</p>
                    <p class="text-xs text-slate-600 dark:text-slate-400">{user.email}</p>
                    {#if user.role === 'admin'}
                        <span
                            class="mt-2 inline-block rounded-full bg-teal-500/20 px-2 py-0.5 text-xs text-teal-700 dark:text-teal-300"
                            >Admin</span
                        >
                    {/if}
                </div>
            {/if}

            <nav class="flex flex-col gap-2">
                <a
                    href="/challenges"
                    class="rounded-lg px-4 py-3 text-sm text-slate-700 transition hover:bg-teal-500/10 hover:text-teal-600 dark:text-slate-300 dark:hover:text-teal-200"
                    onclick={(e) => navigateAndClose('/challenges', e)}
                >
                    Challenges
                </a>
                <a
                    href="/scoreboard"
                    class="rounded-lg px-4 py-3 text-sm text-slate-700 transition hover:bg-teal-500/10 hover:text-teal-600 dark:text-slate-300 dark:hover:text-teal-200"
                    onclick={(e) => navigateAndClose('/scoreboard', e)}
                >
                    Scoreboard
                </a>
                <a
                    href="/users"
                    class="rounded-lg px-4 py-3 text-sm text-slate-700 transition hover:bg-teal-500/10 hover:text-teal-600 dark:text-slate-300 dark:hover:text-teal-200"
                    onclick={(e) => navigateAndClose('/users', e)}
                >
                    Users
                </a>
                <a
                    href="/profile"
                    class="rounded-lg px-4 py-3 text-sm text-slate-700 transition hover:bg-teal-500/10 hover:text-teal-600 dark:text-slate-300 dark:hover:text-teal-200"
                    onclick={(e) => navigateAndClose('/profile', e)}
                >
                    Profile
                </a>
                {#if user?.role === 'admin'}
                    <a
                        href="/admin"
                        class="rounded-lg px-4 py-3 text-sm text-slate-700 transition hover:bg-teal-500/10 hover:text-teal-600 dark:text-slate-300 dark:hover:text-teal-200"
                        onclick={(e) => navigateAndClose('/admin', e)}
                    >
                        Admin
                    </a>
                {/if}
            </nav>

            <div class="my-6 border-t border-slate-200 dark:border-slate-800"></div>

            <div class="flex flex-col gap-3">
                <button
                    class="flex items-center justify-between rounded-lg border border-slate-300 px-4 py-3 text-sm text-slate-700 transition hover:border-teal-500 hover:text-teal-600 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 dark:hover:text-teal-200"
                    onclick={toggleThemeCallback}
                >
                    <span>{theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}</span>
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
                        class="rounded-lg border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700 transition hover:border-red-400 hover:bg-red-100 dark:border-red-900 dark:bg-red-950/50 dark:text-red-400 dark:hover:bg-red-950"
                        onclick={async () => {
                            try {
                                await api.logout()
                            } catch {
                                clearAuthCallback()
                            }
                            navigate('/login')
                            closeMobileMenu()
                        }}
                    >
                        Logout
                    </button>
                {:else}
                    <a
                        href="/login"
                        class="rounded-lg border border-slate-300 px-4 py-3 text-center text-sm text-slate-800 transition hover:border-teal-500 hover:text-teal-600 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 dark:hover:text-teal-200"
                        onclick={(e) => navigateAndClose('/login', e)}
                    >
                        Login
                    </a>
                    <a
                        href="/register"
                        class="rounded-lg bg-teal-500/20 px-4 py-3 text-center text-sm text-teal-700 transition hover:bg-teal-500/30 dark:text-teal-200"
                        onclick={(e) => navigateAndClose('/register', e)}
                    >
                        Register
                    </a>
                {/if}
            </div>
        </div>
    </div>
</aside>
