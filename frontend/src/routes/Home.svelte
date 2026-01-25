<script lang="ts">
    import { authStore } from '../lib/stores'
    import { navigate } from '../lib/router'
</script>

<section class="fade-in">
    <div class="relative overflow-hidden rounded-3xl border border-slate-800/80 bg-slate-900/40 p-10 shadow-glass">
        <div class="absolute inset-0 opacity-40">
            <div class="absolute -top-24 -right-10 h-64 w-64 rounded-full bg-teal-500/20 blur-3xl"></div>
            <div class="absolute -bottom-32 left-10 h-72 w-72 rounded-full bg-emerald-400/10 blur-3xl"></div>
        </div>
        <div class="relative z-10">
            <p class="text-xs uppercase tracking-[0.3em] text-teal-300/70">CTF Command Center</p>
            <h1 class="mt-4 text-4xl font-semibold text-slate-100 md:text-5xl">
                미션을 기록하고,
                <span class="text-teal-200">점수를 증명하세요.</span>
            </h1>
            <p class="mt-4 max-w-2xl text-sm text-slate-300">
                smctf는 팀과 개인이 동시에 집중할 수 있는 CTF 대시보드입니다. 깔끔한 문제 목록, 즉각적인 제출 피드백,
                그리고 타임라인 기반 점수 추적을 한 곳에서 제공합니다.
            </p>
            <div class="mt-8 flex flex-wrap gap-4">
                <a
                    href="/challenges"
                    class="rounded-full bg-teal-500/30 px-6 py-3 text-sm text-teal-100 transition hover:bg-teal-500/40"
                    on:click|preventDefault={() => navigate('/challenges')}>문제 풀기</a
                >
                {#if !$authStore.user}
                    <a
                        href="/register"
                        class="rounded-full border border-slate-700 px-6 py-3 text-sm text-slate-200 transition hover:border-teal-400"
                        on:click|preventDefault={() => navigate('/register')}>새 계정 만들기</a
                    >
                {/if}
            </div>
        </div>
    </div>

    <div class="mt-10 grid gap-6 md:grid-cols-3">
        <div class="rounded-2xl border border-slate-800/80 bg-slate-900/40 p-6">
            <h3 class="text-lg text-slate-100">현황 가시화</h3>
            <p class="mt-2 text-sm text-slate-400">실시간 점수와 해결 시간을 확인하며 팀의 집중도를 관리하세요.</p>
        </div>
        <div class="rounded-2xl border border-slate-800/80 bg-slate-900/40 p-6">
            <h3 class="text-lg text-slate-100">정밀한 에러 처리</h3>
            <p class="mt-2 text-sm text-slate-400">
                API에서 제공하는 세부 오류를 그대로 반영해 제출 실패 원인을 바로 파악합니다.
            </p>
        </div>
        <div class="rounded-2xl border border-slate-800/80 bg-slate-900/40 p-6">
            <h3 class="text-lg text-slate-100">운영 전용 관리</h3>
            <p class="mt-2 text-sm text-slate-400">
                관리자는 새로운 문제를 즉시 생성하고 활성화 상태를 제어할 수 있습니다.
            </p>
        </div>
    </div>
</section>
