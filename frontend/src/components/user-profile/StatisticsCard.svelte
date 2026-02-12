<script lang="ts">
    interface Props {
        totalPoints: number
        solvedCount: number
    }

    let { totalPoints, solvedCount }: Props = $props()
</script>

<div class="mt-8 rounded-2xl border border-border bg-surface p-6">
    <h3 class="text-lg text-text">Statistics</h3>
    <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <div class="rounded-xl border border-border bg-surface-muted p-4">
            <p class="text-xs text-text-muted">Total Points</p>
            <p class="mt-1 text-2xl font-semibold text-text">{totalPoints}</p>
        </div>
        <div class="rounded-xl border border-border bg-surface-muted p-4">
            <p class="text-xs text-text-muted">Problems Solved</p>
            <p class="mt-1 text-2xl font-semibold text-text">{solvedCount}</p>
        </div>
    </div>
</div>
