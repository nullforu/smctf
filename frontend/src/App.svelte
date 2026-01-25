<script lang="ts">
    import { onDestroy, onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, clearAuth, setAuthUser } from './lib/stores'
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

    let currentPath = '/'
    let component = Home
    let booting = true

    const normalizePath = (path: string) => {
        if (path.length > 1 && path.endsWith('/')) {
            return path.replace(/\/+$/, '')
        }
        return path
    }

    const resolvePath = () => normalizePath(window.location.pathname || '/')

    const updateRoute = () => {
        currentPath = resolvePath()
        component = routes[currentPath] ?? NotFound
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

    onDestroy(() => {})
</script>

<div class="min-h-screen grid-overlay">
    <header class="border-b border-slate-800/70 bg-slate-950/70 backdrop-blur">
        <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
            <a href="/" class="flex items-center gap-3" on:click|preventDefault={() => navigate('/')}>
                <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-teal-500/10 text-teal-300">
                    <span class="font-display text-lg">Σ</span>
                </div>
                <div>
                    <p class="font-display text-xl">SMCTF</p>
                    <p class="text-xs text-slate-400">Secure Mission Control</p>
                </div>
            </a>
            <nav class="hidden items-center gap-6 text-sm text-slate-300 md:flex">
                <a
                    class="hover:text-teal-200"
                    href="/challenges"
                    on:click|preventDefault={() => navigate('/challenges')}>Challenges</a
                >
                <a
                    class="hover:text-teal-200"
                    href="/scoreboard"
                    on:click|preventDefault={() => navigate('/scoreboard')}>Scoreboard</a
                >
                <a class="hover:text-teal-200" href="/profile" on:click|preventDefault={() => navigate('/profile')}
                    >Profile</a
                >
                {#if $authStore.user?.role === 'admin'}
                    <a class="hover:text-teal-200" href="/admin" on:click|preventDefault={() => navigate('/admin')}
                        >Admin</a
                    >
                {/if}
            </nav>
            <div class="flex items-center gap-3">
                {#if $authStore.user}
                    <div class="hidden text-right text-xs text-slate-400 sm:block">
                        <p class="text-slate-200">{$authStore.user.username}</p>
                        <p>{$authStore.user.email}</p>
                    </div>
                    <button
                        class="rounded-full border border-slate-700 px-4 py-2 text-xs text-slate-200 transition hover:border-teal-400 hover:text-teal-200"
                        on:click={async () => {
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
                        on:click|preventDefault={() => navigate('/login')}>Login</a
                    >
                    <a
                        href="/register"
                        class="rounded-full bg-teal-500/20 px-4 py-2 text-xs text-teal-200 transition hover:bg-teal-500/30"
                        on:click|preventDefault={() => navigate('/register')}>Register</a
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
            <svelte:component this={component} />
        {/if}
    </main>

    <footer class="border-t border-slate-800/70 py-6 text-center text-xs text-slate-500">
        <p>smctf · Built for rapid CTF operations · API: {import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'}</p>
    </footer>
</div>
