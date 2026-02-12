<script lang="ts">
    interface Props {
        totalPoints: number
        solvedCount: number
    }

    let { totalPoints, solvedCount }: Props = $props()
</script>

<div class="mt-8 rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
    <h3 class="text-lg text-slate-900 dark:text-slate-100">Statistics</h3>
    <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <div class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40">
            <p class="text-xs text-slate-600 dark:text-slate-400">Total Points</p>
            <p class="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">{totalPoints}</p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800/70 dark:bg-slate-950/40">
            <p class="text-xs text-slate-600 dark:text-slate-400">Problems Solved</p>
            <p class="mt-1 text-2xl font-semibold text-slate-900 dark:text-slate-100">{solvedCount}</p>
        </div>
    </div>
</div>
