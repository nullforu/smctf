<script lang="ts">
    import { onMount } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, themeStore, setAuthUser, clearAuth } from './lib/stores'
    import { api } from './lib/api'
    import { configStore, loadConfig } from './lib/config'
    import _Header from './components/Header.svelte'
    import Home from './routes/Home.svelte'
    import Login from './routes/Login.svelte'
    import Register from './routes/Register.svelte'
    import Challenges from './routes/Challenges.svelte'
    import Scoreboard from './routes/Scoreboard.svelte'
    import Teams from './routes/Teams.svelte'
    import TeamProfile from './routes/TeamProfile.svelte'
    import Users from './routes/Users.svelte'
    import UserProfile from './routes/UserProfile.svelte'
    import Admin from './routes/Admin.svelte'
    import NotFound from './routes/NotFound.svelte'

    const Header = _Header

    const routes: Record<string, typeof Home> = {
        '/': Home,
        '/login': Login,
        '/register': Register,
        '/challenges': Challenges,
        '/scoreboard': Scoreboard,
        '/teams': Teams,
        '/profile': UserProfile,
        '/users': Users,
        '/admin': Admin,
    }

    const dynamicRoutes: Array<{
        pattern: RegExp
        component: typeof Home
        extractParams: (path: string) => Record<string, string>
    }> = [
        {
            pattern: /^\/users\/(\d+)$/,
            component: UserProfile,
            extractParams: (path) => {
                const match = path.match(/^\/users\/(\d+)$/)
                return match ? { id: match[1] } : { id: '' }
            },
        },
        {
            pattern: /^\/teams\/(\d+)$/,
            component: TeamProfile,
            extractParams: (path) => {
                const match = path.match(/^\/teams\/(\d+)$/)
                return match ? { id: match[1] } : { id: '' }
            },
        },
    ]

    let currentPath = $state('/')
    let Component = $state(Home)
    let routeParams = $state<Record<string, string>>({})
    let booting = $state(true)
    let auth = $state(get(authStore))
    let theme = $state(get(themeStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    $effect(() => {
        const unsubscribe = configStore.subscribe((value) => {
            if (typeof document !== 'undefined') {
                document.title = value.title || 'SMCTF'
            }
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

        if (routes[currentPath]) {
            Component = routes[currentPath]
            routeParams = {}
        } else {
            let matched = false
            for (const route of dynamicRoutes) {
                if (route.pattern.test(currentPath)) {
                    Component = route.component
                    routeParams = route.extractParams(currentPath)
                    matched = true
                    break
                }
            }
            if (!matched) {
                Component = NotFound
                routeParams = {}
            }
        }
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
        void loadConfig()
        loadSession()
        return () => window.removeEventListener('popstate', updateRoute)
    })
</script>

<div class="min-h-screen">
    <Header user={auth.user} />

    <main class="mx-auto w-full max-w-6xl px-6 py-10">
        {#if booting}
            <div class="rounded-2xl border border-border bg-surface p-8 text-center text-text-muted">
                Checking session...
            </div>
        {:else}
            <Component {routeParams} />
        {/if}
    </main>

    <footer class="border-t border-border py-6 text-center text-xs text-text-subtle">
        <p>Copyright &copy; 2026 Semyeong Computer High School, All rights reserved.</p>
    </footer>
</div>
