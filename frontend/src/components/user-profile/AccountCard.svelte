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

<div class="mt-6 rounded-2xl border border-slate-200 bg-white p-6 dark:border-slate-800/80 dark:bg-slate-900/40">
    <h3 class="text-lg text-slate-900 dark:text-slate-100">Account</h3>

    <div class="mt-4 space-y-2 text-sm text-slate-700 dark:text-slate-300">
        <div class="flex items-center justify-between gap-4">
            <span class="text-slate-600 dark:text-slate-400">Username</span>

            {#if editingUsername}
                <div class="flex items-center gap-2">
                    <input
                        class="rounded-md border border-slate-300 bg-white px-2 py-1 text-sm dark:border-slate-700 dark:bg-slate-900"
                        bind:value={usernameInput}
                        disabled={savingUsername}
                    />
                    <button
                        class="text-sm text-teal-600 hover:underline disabled:opacity-50"
                        disabled={savingUsername}
                        onclick={onSave}
                    >
                        Save
                    </button>
                    <button class="text-sm text-slate-500 hover:underline" onclick={cancelEdit}>Cancel</button>
                </div>
            {:else}
                <div class="flex items-center gap-3">
                    <span>{user.username}</span>
                    <button class="text-xs text-teal-600 hover:underline" onclick={() => (editingUsername = true)}>
                        Edit
                    </button>
                </div>
            {/if}
        </div>

        <div class="flex justify-between">
            <span class="text-slate-600 dark:text-slate-400">Email</span>
            <span>{authEmail}</span>
        </div>

        <div class="flex justify-between">
            <span class="text-slate-600 dark:text-slate-400">Role</span>
            <span class="uppercase text-teal-600 dark:text-teal-200">{user.role}</span>
        </div>
    </div>
</div>
