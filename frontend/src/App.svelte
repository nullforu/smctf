<script lang="ts">
    import { onDestroy, onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, clearAuth, setAuthUser, type AuthState } from './lib/stores'
    import { api } from './lib/api'
    import { navigate } from './lib/router'
    import Home from './routes/Home.svelte'
    import Login from './routes/Login.svelte'
    import Register from './routes/Register.svelte'
    import Challenges from './routes/Challenges.svelte'
    import Scoreboard from './routes/Scoreboard.svelte'
    import Profile from './routes/Profile.svelte'
    import Admin from './routes/Admin.svelte'
    import NotFound from './routes/NotFound.svelte'

    const routes: Record<string, typeof Home> = {
        '/': Home,
        '/login': Login,
        '/register': Register,
        '/challenges': Challenges,
        '/scoreboard': Scoreboard,
        '/profile': Profile,
        '/admin': Admin,
    }

    let currentPath = $state('/')
    let Component = $state(Home)
    let booting = $state(true)
    let auth = $state<AuthState>(get(authStore))
    const unsubscribe = authStore.subscribe((value) => {
        auth = value
    })

    const normalizePath = (path: string) => {
        if (path.length > 1 && path.endsWith('/')) {
            return path.replace(/\/+$/, '')
        }
        return path
    }

    const resolvePath = () => normalizePath(window.location.pathname || '/')

    const updateRoute = () => {
        currentPath = resolvePath()
        Component = routes[currentPath] ?? NotFound
    }

    const onNav = (event: MouseEvent, path: string) => {
        event.preventDefault()
        navigate(path)
    }

    const loadSession = async () => {
        const { accessToken } = get(authStore)
        if (!accessToken) {
            booting = false
            return
        }
        try {
            const user = await api.me()
            setAuthUser(user)
        } catch {
            clearAuth()
        } finally {
            booting = false
        }
    }

    onMount(() => {
        updateRoute()
        const handler = () => updateRoute()
        window.addEventListener('popstate', handler)
        loadSession()
        return () => window.removeEventListener('popstate', handler)
    })

    onDestroy(unsubscribe)
</script>

<div class="min-h-screen grid-overlay">
    <header class="border-b border-slate-800/70 bg-slate-950/70 backdrop-blur">
        <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
            <a href="/" class="flex items-center gap-3" onclick={(event) => onNav(event, '/')}>
                <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-teal-500/10 text-teal-300">
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
                    <p class="font-display text-xl">CTF</p>
                    <p class="text-xs text-slate-400">Capture The Flag</p>
                </div>
            </a>
            <nav class="hidden items-center gap-6 text-sm text-slate-300 md:flex">
                <a class="hover:text-teal-200" href="/challenges" onclick={(event) => onNav(event, '/challenges')}
                    >Challenges</a
                >
                <a class="hover:text-teal-200" href="/scoreboard" onclick={(event) => onNav(event, '/scoreboard')}
                    >Scoreboard</a
                >
                <a class="hover:text-teal-200" href="/profile" onclick={(event) => onNav(event, '/profile')}>Profile</a>
                {#if auth.user?.role === 'admin'}
                    <a class="hover:text-teal-200" href="/admin" onclick={(event) => onNav(event, '/admin')}>Admin</a>
                {/if}
            </nav>
            <div class="flex items-center gap-3">
                {#if auth.user}
                    <div class="hidden text-right text-xs text-slate-400 sm:block">
                        <p class="text-slate-200">{auth.user.username}</p>
                        <p>{auth.user.email}</p>
                    </div>
                    <button
                        class="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-200 transition hover:border-teal-400 hover:text-teal-200"
                        onclick={async () => {
                            try {
                                await api.logout()
                            } catch {
                                clearAuth()
                            }
                            navigate('/login')
                        }}
                    >
                        Logout
                    </button>
                {:else}
                    <a
                        href="/login"
                        class="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-200 transition hover:border-teal-400 hover:text-teal-200"
                        onclick={(event) => onNav(event, '/login')}>Login</a
                    >
                    <a
                        href="/register"
                        class="rounded-full bg-teal-500/20 px-4 py-2 text-xs text-teal-200 transition hover:bg-teal-500/30"
                        onclick={(event) => onNav(event, '/register')}>Register</a
                    >
                {/if}
            </div>
        </div>
    </header>

    <main class="mx-auto w-full max-w-6xl px-6 py-10">
        {#if booting}
            <div class="rounded-2xl border border-slate-800/70 bg-slate-900/40 p-8 text-center text-slate-400">
                세션 확인 중...
            </div>
        {:else}
            <Component />
        {/if}
    </main>

    <footer class="border-t border-slate-800/70 py-6 text-center text-xs text-slate-500">
        <p>Copyright &copy; 2026 Semyeong Computer High School, All rights reserved.</p>
    </footer>
</div>
