<script lang="ts">
    import { navigate } from '../lib/router'

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()
</script>

<section class="fade-in">
    <div
        class="rounded-3xl border border-slate-200 bg-white p-10 text-center dark:border-slate-800/80 dark:bg-slate-900/40"
    >
        <h2 class="text-3xl text-slate-900 dark:text-slate-100">404</h2>
        <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">페이지를 찾을 수 없습니다.</p>
        <a
            href="/"
            class="mt-6 inline-flex rounded-full bg-teal-600 px-6 py-2 text-sm text-white transition hover:bg-teal-700 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
            onclick={(event) => {
                event.preventDefault()
                navigate('/')
            }}>홈으로</a
        >
    </div>
</section>
