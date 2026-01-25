<script lang="ts">
    import { onDestroy } from 'svelte'
    import { get } from 'svelte/store'
    import { authStore, type AuthState } from '../lib/stores'
    import { navigate } from '../lib/router'

    let auth = $state<AuthState>(get(authStore))

    onDestroy(
        authStore.subscribe((value) => {
            auth = value
        }),
    )

    const onNav = (event: MouseEvent, path: string) => {
        event.preventDefault()
        navigate(path)
    }
</script>

<section class="fade-in">
    <div class="relative overflow-hidden rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10 shadow-glass">
        <div class="absolute inset-0 opacity-40">
            <div class="absolute -top-24 -right-10 h-64 w-64 rounded-full bg-teal-500/20 blur-3xl"></div>
            <div class="absolute -bottom-32 left-10 h-72 w-72 rounded-full bg-emerald-400/10 blur-3xl"></div>
        </div>
        <div class="relative z-10">
            <p class="text-xs uppercase tracking-[0.3em] text-teal-300/70">CTF Command Center</p>
            <h1 class="mt-4 text-xl font-semibold text-slate-100 md:text-3xl">SMCTF에 오신 것을 환영합니다.</h1>
            <p class="mt-4 max-w-2xl text-sm text-slate-300">
                <a href="https://github.com/nullforu/smctf">이곳</a>에서 SMCTF 플랫폼을 구성하는 방법을 확인할 수
                있습니다.
            </p>
            <div class="mt-8 flex flex-wrap gap-4">
                <a
                    href="/challenges"
                    class="rounded-full bg-teal-500/30 px-6 py-3 text-sm text-teal-100 transition hover:bg-teal-500/40"
                    onclick={(event) => onNav(event, '/challenges')}>문제 풀기</a
                >
                {#if !auth.user}
                    <a
                        href="/register"
                        class="rounded-full border border-slate-700 px-6 py-3 text-sm text-slate-200 transition hover:border-teal-400"
                        onclick={(event) => onNav(event, '/register')}>가입하기</a
                    >
                {/if}
            </div>
        </div>
    </div>
</section>
