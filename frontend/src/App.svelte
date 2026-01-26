<script lang="ts">
    import { onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, themeStore, setAuthUser, clearAuth } from './lib/stores'
    import { api } from './lib/api'
    import Header from './components/Header.svelte'
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
    let auth = $state(get(authStore))
    let theme = $state(get(themeStore))

    const HeaderComponent = Header

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    $effect(() => {
        const unsubscribe = themeStore.subscribe((value) => {
            theme = value
            if (typeof document !== 'undefined') {
                if (value === 'dark') {
                    document.documentElement.classList.add('dark')
                } else {
                    document.documentElement.classList.remove('dark')
                }
            }
        })
        return unsubscribe
    })

    const normalizePath = (path: string) => {
        return path.length > 1 && path.endsWith('/') ? path.replace(/\/+$/, '') : path
    }

    const updateRoute = () => {
        currentPath = normalizePath(window.location.pathname || '/')
        Component = routes[currentPath] ?? NotFound
    }

    const loadSession = async () => {
        if (!auth.accessToken) {
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
        window.addEventListener('popstate', updateRoute)
        loadSession()
        return () => window.removeEventListener('popstate', updateRoute)
    })
</script>

<div class="min-h-screen">
    <HeaderComponent user={auth.user} />

    <main class="mx-auto w-full max-w-6xl px-6 py-10">
        {#if booting}
            <div
                class="rounded-2xl border border-slate-200 bg-white p-8 text-center text-slate-600 dark:border-slate-800/70 dark:bg-slate-900/40 dark:text-slate-400"
            >
                세션 확인 중...
            </div>
        {:else}
            <Component />
        {/if}
    </main>

    <footer class="border-t border-slate-200 py-6 text-center text-xs text-slate-500 dark:border-slate-800/70">
        <p>Copyright &copy; 2026 Semyeong Computer High School, All rights reserved.</p>
    </footer>
</div>
