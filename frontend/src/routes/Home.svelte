<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { navigate } from '../lib/router'
    import { configStore } from '../lib/config'
    import Markdown from '../components/Markdown.svelte'
    import { t } from '../lib/i18n'
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

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()
</script>

<section class="fade-in">
    <div class="relative overflow-hidden p-4 sm:p-8 md:p-10">
        <div class="relative z-10">
            <h1 class="mt-2 text-2xl font-semibold text-text sm:mt-4 md:text-3xl lg:text-4xl">
                {appConfig.title}
            </h1>
            <div class="mt-3 max-w-2xl text-base text-text sm:mt-4 sm:text-base md:text-lg">
                <Markdown content={appConfig.description} />
            </div>
            <div class="mt-6 flex flex-wrap gap-3 sm:mt-8 sm:gap-4">
                <a
                    href="/challenges"
                    class="rounded-full bg-accent px-5 py-2.5 text-sm text-contrast-foreground transition hover:bg-accent-strong sm:px-6 sm:py-3 sm:text-base"
                    onclick={(e) => navigate('/challenges', e)}>{$t('home.ctaChallenges')}</a
                >
                {#if !auth.user}
                    <a
                        href="/register"
                        class="rounded-full border border-border px-5 py-2.5 text-sm text-text transition hover:border-accent sm:px-6 sm:py-3 sm:text-base"
                        onclick={(e) => navigate('/register', e)}>{$t('home.ctaSignUp')}</a
                    >
                {/if}
            </div>
        </div>
    </div>
</section>
