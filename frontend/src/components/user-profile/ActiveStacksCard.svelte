<script lang="ts">
    import type { Stack } from '../../lib/types'

    interface Props {
        activeStacks: Stack[]
        stacksError: string
        stacksLoading: boolean
        stackDeletingId: number | null
        onRefresh: () => void
        onDelete: (challengeId: number) => void
        formatOptionalDateTime: (value?: string | null) => string
    }

    let {
        activeStacks,
        stacksError,
        stacksLoading,
        stackDeletingId,
        onRefresh,
        onDelete,
        formatOptionalDateTime,
    }: Props = $props()
</script>

<div class="mt-6 rounded-2xl border border-border bg-surface p-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
        <h3 class="text-lg text-text">Active Stacks</h3>
        <button
            class="text-xs uppercase tracking-wide text-text-subtle hover:text-text disabled:opacity-60"
            onclick={onRefresh}
            disabled={stacksLoading}
        >
            {stacksLoading ? 'Loading...' : 'Refresh'}
        </button>
    </div>

    {#if stacksError}
        <p class="mt-4 rounded-xl border border-danger/40 bg-danger/10 px-4 py-2 text-xs text-danger">
            {stacksError}
        </p>
    {:else if activeStacks.length === 0}
        <div class="mt-4 rounded-xl border border-border bg-surface-muted p-5 text-center">
            <p class="text-sm text-text-muted">No active stacks.</p>
        </div>
    {:else}
        <div class="mt-4 space-y-3">
            {#each activeStacks as stack}
                <div class="rounded-xl border border-border bg-surface-muted p-5">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <p class="text-sm font-medium text-text">
                                Challenge #{stack.challenge_id}
                            </p>
                            <p class="mt-1 text-xs text-text-subtle">Status: {stack.status}</p>
                        </div>
                        <div class="flex flex-wrap items-center gap-3 text-xs text-text-muted">
                            <span>
                                {stack.node_public_ip && stack.node_port
                                    ? `${stack.node_public_ip}:${stack.node_port}`
                                    : 'Pending'}
                            </span>
                            <button
                                class="rounded-lg border border-danger/30 px-3 py-1.5 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60"
                                type="button"
                                onclick={() => onDelete(stack.challenge_id)}
                                disabled={stackDeletingId === stack.challenge_id || stacksLoading}
                            >
                                {stackDeletingId === stack.challenge_id ? 'Deleting...' : 'Delete'}
                            </button>
                        </div>
                    </div>
                    <div class="mt-2 text-xs text-text-subtle">
                        TTL: {formatOptionalDateTime(stack.ttl_expires_at)}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
