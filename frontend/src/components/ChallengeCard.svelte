<script lang="ts">
    import type { Challenge } from '../lib/types'

    interface Props {
        challenge: Challenge
        isSolved: boolean
        onClick: () => void
    }

    let { challenge, isSolved, onClick }: Props = $props()
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
    class={`rounded-2xl border p-6 transition cursor-pointer hover:shadow-lg ${
        challenge.is_active
            ? 'border-slate-200 bg-white dark:border-slate-800/80 dark:bg-slate-900/40 hover:border-teal-500 dark:hover:border-teal-400'
            : 'border-slate-300/40 bg-slate-50 opacity-60 dark:border-slate-800/40 dark:bg-slate-900/20'
    }`}
    onclick={onClick}
>
    <div class="flex items-start justify-between">
        <div class="flex-1">
            <h3 class="text-lg font-medium text-slate-900 dark:text-slate-100">{challenge.title}</h3>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
                <span
                    class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200"
                    >{challenge.category}</span
                >
                <span class="text-slate-600 dark:text-slate-400">{challenge.points} pts</span>
            </div>
        </div>
        {#if isSolved}
            <span class="rounded-full bg-emerald-500/20 px-3 py-1 text-xs text-emerald-700 dark:text-emerald-200"
                >Solved</span
            >
        {:else if !challenge.is_active}
            <span
                class="rounded-full bg-slate-400/10 px-3 py-1 text-xs text-slate-600 dark:bg-slate-500/10 dark:text-slate-300"
                >Inactive</span
            >
        {/if}
    </div>
</div>
