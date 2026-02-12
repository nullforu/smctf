<script lang="ts">
    import type { UserDetail } from '../../lib/types'

    interface Props {
        user: UserDetail
    }

    let { user }: Props = $props()

    const roleClasses = (role: string) =>
        role === 'admin'
            ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
            : 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300'
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
    <div>
        <h2 class="text-3xl text-slate-900 dark:text-slate-100">{user.username}</h2>
        <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">User ID: {user.id}</p>
        <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">Team: {user.team_name}</p>
    </div>
    <span
        class={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium uppercase ${roleClasses(user.role)}`}
    >
        {user.role}
    </span>
</div>
