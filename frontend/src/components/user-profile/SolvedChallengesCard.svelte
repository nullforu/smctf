<script lang="ts">
    import type { SolvedChallenge } from '../../lib/types'

    interface Props {
        solved: SolvedChallenge[]
        formatDateTime: (value: string) => string
    }

    let { solved, formatDateTime }: Props = $props()
</script>

<div class="mt-8 rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
    <div class="flex items-center justify-between">
        <h3 class="text-lg text-slate-900 dark:text-slate-100">Solved Challenges</h3>
        <span class="text-sm text-slate-600 dark:text-slate-400">
            {solved.length}
            {solved.length === 1 ? 'problem' : 'problems'}
        </span>
    </div>

    <div class="mt-6 space-y-3">
        {#each solved as item}
            <div
                class="rounded-xl border border-slate-200 bg-slate-50 p-5 dark:border-slate-800/70 dark:bg-slate-950/40"
            >
                <h4 class="text-base font-medium text-slate-900 dark:text-slate-100">
                    {item.title}
                    <span class="ml-2 text-xs text-teal-600">{item.points} pts</span>
                </h4>
                <p class="mt-2 text-sm text-slate-600 dark:text-slate-400">
                    Solved at {formatDateTime(item.solved_at)}
                </p>
            </div>
        {/each}

        {#if solved.length === 0}
            <div
                class="rounded-xl border border-slate-200 bg-slate-50 p-8 text-center dark:border-slate-800/70 dark:bg-slate-950/40"
            >
                <p class="text-sm text-slate-600 dark:text-slate-400">No challenges solved yet.</p>
            </div>
        {/if}
    </div>
</div>
