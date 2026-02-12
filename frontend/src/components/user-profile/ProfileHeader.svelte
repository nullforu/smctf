<script lang="ts">
    import type { UserDetail } from '../../lib/types'

    interface Props {
        user: UserDetail
    }

    let { user }: Props = $props()

    const roleClasses = (role: string) =>
        role === 'admin' ? 'bg-secondary/20 text-secondary  ' : 'bg-accent/20 text-accent-strong  '
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
    <div>
        <h2 class="text-3xl text-text">{user.username}</h2>
        <p class="mt-1 text-sm text-text-muted">User ID: {user.id}</p>
        <p class="mt-1 text-sm text-text-muted">Team: {user.team_name}</p>
    </div>
    <span
        class={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium uppercase ${roleClasses(user.role)}`}
    >
        {user.role}
    </span>
</div>
