<script lang="ts">
    import { navigate } from '../lib/router'

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()
</script>

<section class="fade-in">
    <div class="rounded-3xl border border-border bg-surface p-10 text-center">
        <h2 class="text-3xl text-text">404</h2>
        <p class="mt-2 text-sm text-text-muted">페이지를 찾을 수 없습니다.</p>
        <a
            href="/"
            class="mt-6 inline-flex rounded-full bg-accent px-6 py-2 text-sm text-contrast-foreground transition hover:bg-accent-strong"
            onclick={(event) => {
                event.preventDefault()
                navigate('/')
            }}>홈으로</a
        >
    </div>
</section>
