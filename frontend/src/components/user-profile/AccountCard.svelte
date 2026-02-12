<script lang="ts">
    import type { UserDetail } from '../../lib/types'
    import { getRoleKey, t } from '../../lib/i18n'

    interface Props {
        user: UserDetail
        authEmail?: string
        savingUsername: boolean
        onSave: () => void
        editingUsername?: boolean
        usernameInput?: string
    }

    let {
        user,
        authEmail,
        savingUsername,
        onSave,
        editingUsername = $bindable(false),
        usernameInput = $bindable(''),
    }: Props = $props()

    const cancelEdit = () => {
        editingUsername = false
        usernameInput = user.username
    }
</script>

<div class="mt-6 rounded-2xl border border-border bg-surface p-6">
    <h3 class="text-lg text-text">{$t('profile.account')}</h3>

    <div class="mt-4 space-y-2 text-sm text-text">
        <div class="flex items-center justify-between gap-4">
            <span class="text-text-muted">{$t('common.username')}</span>

            {#if editingUsername}
                <div class="flex items-center gap-2">
                    <input
                        class="rounded-md border border-border bg-surface px-2 py-1 text-sm"
                        bind:value={usernameInput}
                        disabled={savingUsername}
                    />
                    <button
                        class="text-sm text-accent hover:underline disabled:opacity-50"
                        disabled={savingUsername}
                        onclick={onSave}
                    >
                        {$t('profile.save')}
                    </button>
                    <button class="text-sm text-text-subtle hover:underline" onclick={cancelEdit}>
                        {$t('profile.cancel')}
                    </button>
                </div>
            {:else}
                <div class="flex items-center gap-3">
                    <span>{user.username}</span>
                    <button class="text-xs text-accent hover:underline" onclick={() => (editingUsername = true)}>
                        {$t('profile.edit')}
                    </button>
                </div>
            {/if}
        </div>

        <div class="flex justify-between">
            <span class="text-text-muted">{$t('common.email')}</span>
            <span>{authEmail}</span>
        </div>

        <div class="flex justify-between">
            <span class="text-text-muted">{$t('common.role')}</span>
            <span class="uppercase text-accent">{$t(getRoleKey(user.role))}</span>
        </div>
    </div>
</div>
