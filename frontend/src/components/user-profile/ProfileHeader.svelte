<script lang="ts">
    import type { UserDetail } from '../../lib/types'
    import { getRoleKey, t } from '../../lib/i18n'

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
        <p class="mt-1 text-sm text-text-muted">{$t('profile.userId', { id: user.id })}</p>
        <p class="mt-1 text-sm text-text-muted">{$t('profile.team', { team: user.team_name })}</p>
    </div>
    <span
        class={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium uppercase ${roleClasses(user.role)}`}
    >
        {$t(getRoleKey(user.role))}
    </span>
</div>
