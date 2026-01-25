<script lang="ts">
    import { api } from '../lib/api'
    import { clearAuth } from '../lib/stores'
    import { navigate } from '../lib/router'
    import type { AuthUser } from '../lib/types'

    interface Props {
        user: AuthUser | null
    }

    let { user }: Props = $props()
</script>

<header class="border-b border-slate-800/70 bg-slate-950/70 backdrop-blur">
    <div class="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <a href="/" class="flex items-center gap-3" onclick={(event) => navigate('/', event)}>
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
            <a class="hover:text-teal-200" href="/challenges" onclick={(e) => navigate('/challenges', e)}>Challenges</a>
            <a class="hover:text-teal-200" href="/scoreboard" onclick={(e) => navigate('/scoreboard', e)}>Scoreboard</a>
            <a class="hover:text-teal-200" href="/profile" onclick={(e) => navigate('/profile', e)}>Profile</a>
            {#if user?.role === 'admin'}
                <a class="hover:text-teal-200" href="/admin" onclick={(e) => navigate('/admin', e)}>Admin</a>
            {/if}
        </nav>
        <div class="flex items-center gap-3">
            {#if user}
                <div class="hidden text-right text-xs text-slate-400 sm:block">
                    <p class="text-slate-200">{user.username}</p>
                    <p>{user.email}</p>
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
                    onclick={(e) => navigate('/login', e)}>Login</a
                >
                <a
                    href="/register"
                    class="rounded-full bg-teal-500/20 px-4 py-2 text-xs text-teal-200 transition hover:bg-teal-500/30"
                    onclick={(e) => navigate('/register', e)}>Register</a
                >
            {/if}
        </div>
    </div>
</header>
