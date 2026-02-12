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
            ? 'border-border bg-surface   hover:border-accent '
            : 'border-border/40 bg-surface-muted opacity-60  '
    }`}
    onclick={onClick}
>
    <div class="flex items-start justify-between">
        <div class="flex-1">
            <h3 class="text-lg font-medium text-text">{challenge.title}</h3>
            <div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
                <span class="rounded-full bg-surface-subtle px-2.5 py-0.5 text-xs font-medium text-text"
                    >{challenge.category}</span
                >
                <span class="text-text-muted">{challenge.points} pts</span>
            </div>
        </div>
        {#if isSolved}
            <span class="rounded-full bg-success/20 px-3 py-1 text-xs text-success">Solved</span>
        {:else if !challenge.is_active}
            <span class="rounded-full bg-surface/10 px-3 py-1 text-xs text-text-muted">Inactive</span>
        {/if}
    </div>
</div>
