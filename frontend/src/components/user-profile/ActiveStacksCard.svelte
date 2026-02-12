<script lang="ts">
    import type { Stack } from '../../lib/types'
    import { t } from '../../lib/i18n'

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
        <h3 class="text-lg text-text">{$t('profile.activeStacks')}</h3>
        <button
            class="text-xs uppercase tracking-wide text-text-subtle hover:text-text disabled:opacity-60"
            onclick={onRefresh}
            disabled={stacksLoading}
        >
            {stacksLoading ? $t('common.loading') : $t('common.refresh')}
        </button>
    </div>

    {#if stacksError}
        <p class="mt-4 rounded-xl border border-danger/40 bg-danger/10 px-4 py-2 text-xs text-danger">
            {stacksError}
        </p>
    {:else if activeStacks.length === 0}
        <div class="mt-4 rounded-xl border border-border bg-surface-muted p-5 text-center">
            <p class="text-sm text-text-muted">{$t('profile.noActiveStacks')}</p>
        </div>
    {:else}
        <div class="mt-4 space-y-3">
            {#each activeStacks as stack}
                <div class="rounded-xl border border-border bg-surface-muted p-5">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <p class="text-sm font-medium text-text">
                                {$t('profile.challengeLabel', { id: stack.challenge_id })}
                            </p>
                            <p class="mt-1 text-xs text-text-subtle">
                                {$t('profile.statusLabel', { status: stack.status })}
                            </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-3 text-xs text-text-muted">
                            <span>
                                {stack.node_public_ip && stack.node_port
                                    ? `${stack.node_public_ip}:${stack.node_port}`
                                    : $t('common.pending')}
                            </span>
                            <button
                                class="rounded-lg border border-danger/30 px-3 py-1.5 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60"
                                type="button"
                                onclick={() => onDelete(stack.challenge_id)}
                                disabled={stackDeletingId === stack.challenge_id || stacksLoading}
                            >
                                {stackDeletingId === stack.challenge_id ? $t('profile.deleting') : $t('profile.delete')}
                            </button>
                        </div>
                    </div>
                    <div class="mt-2 text-xs text-text-subtle">
                        {$t('profile.ttlLabel', { time: formatOptionalDateTime(stack.ttl_expires_at) })}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
