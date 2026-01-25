<script lang="ts">
    import { navigate } from '../lib/router'
</script>

<section class="fade-in">
    <div class="rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10 text-center">
        <h2 class="text-3xl text-slate-100">404</h2>
        <p class="mt-2 text-sm text-slate-400">페이지를 찾을 수 없습니다.</p>
        <a
            href="/"
            class="mt-6 inline-flex rounded-full bg-teal-500/30 px-6 py-2 text-sm text-teal-100 transition hover:bg-teal-500/40"
            on:click|preventDefault={() => navigate('/')}>홈으로</a
        >
    </div>
</section>
