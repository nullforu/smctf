<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { UserListItem } from '../lib/types'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'
    import { getRoleKey, t } from '../lib/i18n'

    let users: UserListItem[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let searchQuery = $state('')

    const loadUsers = async () => {
        loading = true
        errorMessage = ''

        try {
            users = await api.users()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const normalizedQuery = $derived(searchQuery.trim().toLowerCase())
    const sortedUsers = $derived([...users].sort((a, b) => a.id - b.id))
    const filteredUsers = $derived(
        normalizedQuery
            ? sortedUsers.filter(
                  (user) =>
                      user.username.toLowerCase().includes(normalizedQuery) ||
                      user.id.toString().includes(normalizedQuery) ||
                      user.team_name.toLowerCase().includes(normalizedQuery),
              )
            : sortedUsers,
    )

    onMount(loadUsers)

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-text">{$t('users.title')}</h2>
        </div>
    </div>

    <div class="mt-6">
        <input
            type="text"
            placeholder={$t('users.searchPlaceholder')}
            bind:value={searchQuery}
            class="w-full rounded-xl border border-border bg-surface px-4 py-2.5 text-sm text-text placeholder-text-subtle transition focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
        />
    </div>

    {#if loading}
        <p class="mt-6 text-sm text-text-muted">{$t('common.loading')}</p>
    {:else if errorMessage}
        <p class="mt-6 text-sm text-danger">{errorMessage}</p>
    {:else}
        <div class="mt-6">
            <div class="overflow-hidden rounded-2xl border border-border bg-surface">
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead class="border-b border-border bg-surface-muted">
                            <tr>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.id')}
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.username')}
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.team')}
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.role')}
                                </th>
                                <th
                                    class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.action')}
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                            {#each filteredUsers as user}
                                <tr
                                    class="transition hover:bg-surface-muted cursor-pointer"
                                    onclick={() => navigate(`/users/${user.id}`)}
                                >
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {user.id}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {user.username}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {user.team_name}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm">
                                        <span
                                            class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase {user.role ===
                                            'admin'
                                                ? 'bg-secondary/20 text-secondary '
                                                : 'bg-accent/20 text-accent-strong '}"
                                        >
                                            {$t(getRoleKey(user.role))}
                                        </span>
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                        <button
                                            class="text-accent hover:text-accent-strong"
                                            onclick={(event) => {
                                                event.stopPropagation()
                                                navigate(`/users/${user.id}`)
                                            }}
                                            type="button"
                                        >
                                            {$t('common.view')}
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                            {#if filteredUsers.length === 0}
                                <tr>
                                    <td colspan="5" class="px-6 py-8 text-center text-sm text-text-muted">
                                        {searchQuery ? $t('users.noResults') : $t('users.noUsers')}
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                </div>
            </div>

            {#if filteredUsers.length > 0}
                <p class="mt-4 text-sm text-text-muted">
                    {filteredUsers.length === 1
                        ? $t('users.countSingular', { count: filteredUsers.length })
                        : $t('users.countPlural', { count: filteredUsers.length })}
                    {#if searchQuery}
                        {$t('common.outOf', { total: users.length })}
                    {/if}
                </p>
            {/if}
        </div>
    {/if}
</section>
