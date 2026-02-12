<script lang="ts">
    import type { SolvedChallenge } from '../../lib/types'

    interface Props {
        solved: SolvedChallenge[]
        formatDateTime: (value: string) => string
    }

    let { solved, formatDateTime }: Props = $props()
</script>

<div class="mt-8 rounded-2xl border border-border bg-surface p-6">
    <div class="flex items-center justify-between">
        <h3 class="text-lg text-text">Solved Challenges</h3>
        <span class="text-sm text-text-muted">
            {solved.length}
            {solved.length === 1 ? 'problem' : 'problems'}
        </span>
    </div>

    <div class="mt-6 space-y-3">
        {#each solved as item}
            <div class="rounded-xl border border-border bg-surface-muted p-5">
                <h4 class="text-base font-medium text-text">
                    {item.title}
                    <span class="ml-2 text-xs text-accent">{item.points} pts</span>
                </h4>
                <p class="mt-2 text-sm text-text-muted">
                    Solved at {formatDateTime(item.solved_at)}
                </p>
            </div>
        {/each}

        {#if solved.length === 0}
            <div class="rounded-xl border border-border bg-surface-muted p-8 text-center">
                <p class="text-sm text-text-muted">No challenges solved yet.</p>
            </div>
        {/if}
    </div>
</div>
