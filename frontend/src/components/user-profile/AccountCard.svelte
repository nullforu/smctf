<script lang="ts">
    import type { UserDetail } from '../../lib/types'

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
    <h3 class="text-lg text-text">Account</h3>

    <div class="mt-4 space-y-2 text-sm text-text">
        <div class="flex items-center justify-between gap-4">
            <span class="text-text-muted">Username</span>

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
                        Save
                    </button>
                    <button class="text-sm text-text-subtle hover:underline" onclick={cancelEdit}>Cancel</button>
                </div>
            {:else}
                <div class="flex items-center gap-3">
                    <span>{user.username}</span>
                    <button class="text-xs text-accent hover:underline" onclick={() => (editingUsername = true)}>
                        Edit
                    </button>
                </div>
            {/if}
        </div>

        <div class="flex justify-between">
            <span class="text-text-muted">Email</span>
            <span>{authEmail}</span>
        </div>

        <div class="flex justify-between">
            <span class="text-text-muted">Role</span>
            <span class="uppercase text-accent">{user.role}</span>
        </div>
    </div>
</div>
