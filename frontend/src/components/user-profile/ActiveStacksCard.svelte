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

<div class="mt-6 rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
    <div class="flex flex-wrap items-center justify-between gap-4">
        <h3 class="text-lg text-slate-900 dark:text-slate-100">Active Stacks</h3>
        <button
            class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white disabled:opacity-60"
            onclick={onRefresh}
            disabled={stacksLoading}
        >
            {stacksLoading ? 'Loading...' : 'Refresh'}
        </button>
    </div>

    {#if stacksError}
        <p
            class="mt-4 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
        >
            {stacksError}
        </p>
    {:else if activeStacks.length === 0}
        <div
            class="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-5 text-center dark:border-slate-800/70 dark:bg-slate-950/40"
        >
            <p class="text-sm text-slate-600 dark:text-slate-400">No active stacks.</p>
        </div>
    {:else}
        <div class="mt-4 space-y-3">
            {#each activeStacks as stack}
                <div
                    class="rounded-xl border border-slate-200 bg-slate-50 p-5 dark:border-slate-800/70 dark:bg-slate-950/40"
                >
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <p class="text-sm font-medium text-slate-900 dark:text-slate-100">
                                Challenge #{stack.challenge_id}
                            </p>
                            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">Status: {stack.status}</p>
                        </div>
                        <div class="flex flex-wrap items-center gap-3 text-xs text-slate-600 dark:text-slate-400">
                            <span>
                                {stack.node_public_ip && stack.node_port
                                    ? `${stack.node_public_ip}:${stack.node_port}`
                                    : 'Pending'}
                            </span>
                            <button
                                class="rounded-lg border border-rose-200 px-3 py-1.5 text-xs font-medium text-rose-700 transition hover:border-rose-300 hover:text-rose-800 disabled:opacity-60 dark:border-rose-500/40 dark:text-rose-200 dark:hover:border-rose-400"
                                type="button"
                                onclick={() => onDelete(stack.challenge_id)}
                                disabled={stackDeletingId === stack.challenge_id || stacksLoading}
                            >
                                {stackDeletingId === stack.challenge_id ? 'Deleting...' : 'Delete'}
                            </button>
                        </div>
                    </div>
                    <div class="mt-2 text-xs text-slate-500 dark:text-slate-400">
                        TTL: {formatOptionalDateTime(stack.ttl_expires_at)}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
