<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { navigate } from '../lib/router'
    import { configStore } from '../lib/config'
    import Markdown from '../components/Markdown.svelte'
    let auth = $state(get(authStore))
    let appConfig = $state(get(configStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    $effect(() => {
        const unsubscribe = configStore.subscribe((value) => {
            appConfig = value
        })
        return unsubscribe
    })
</script>

<section class="fade-in">
    <div class="relative overflow-hidden p-4 sm:p-8 md:p-10">
        <div class="relative z-10">
            <h1 class="mt-2 text-2xl font-semibold text-slate-950 dark:text-slate-100 sm:mt-4 md:text-3xl lg:text-4xl">
                {appConfig.title}
            </h1>
            <div class="mt-3 max-w-2xl text-base text-slate-800 dark:text-slate-300 sm:mt-4 sm:text-base md:text-lg">
                <Markdown content={appConfig.description} />
            </div>
            <div class="mt-6 flex flex-wrap gap-3 sm:mt-8 sm:gap-4">
                <a
                    href="/challenges"
                    class="rounded-full bg-teal-600 px-5 py-2.5 text-sm text-white transition hover:bg-teal-700 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40 sm:px-6 sm:py-3 sm:text-base"
                    onclick={(e) => navigate('/challenges', e)}>Challenges</a
                >
                {#if !auth.user}
                    <a
                        href="/register"
                        class="rounded-full border border-slate-300 px-5 py-2.5 text-sm text-slate-700 transition hover:border-teal-500 dark:border-slate-700 dark:text-slate-200 dark:hover:border-teal-400 sm:px-6 sm:py-3 sm:text-base"
                        onclick={(e) => navigate('/register', e)}>Sign Up</a
                    >
                {/if}
            </div>
        </div>
    </div>
</section>
