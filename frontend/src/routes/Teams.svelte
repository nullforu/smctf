<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { TeamSummary } from '../lib/types'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'
    import { t } from '../lib/i18n'

    let teams: TeamSummary[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let searchQuery = $state('')

    const loadTeams = async () => {
        loading = true
        errorMessage = ''

        try {
            teams = await api.teams()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const normalizedQuery = $derived(searchQuery.trim().toLowerCase())
    const sortedTeams = $derived([...teams].sort((a, b) => a.id - b.id))
    const filteredTeams = $derived(
        normalizedQuery
            ? sortedTeams.filter(
                  (team) =>
                      team.name.toLowerCase().includes(normalizedQuery) || team.id.toString().includes(normalizedQuery),
              )
            : sortedTeams,
    )

    onMount(loadTeams)
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-text">{$t('teams.title')}</h2>
        </div>
    </div>

    <div class="mt-6">
        <input
            type="text"
            placeholder={$t('teams.searchPlaceholder')}
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
                                    {$t('common.team')}
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.members')}
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.totalScore')}
                                </th>
                                <th
                                    class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    {$t('common.action')}
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                            {#each filteredTeams as team}
                                <tr
                                    class="transition hover:bg-surface-muted cursor-pointer"
                                    onclick={() => navigate(`/teams/${team.id}`)}
                                >
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {team.id}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {team.name}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                        {team.member_count}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-accent">
                                        {$t('common.pointsShort', { points: team.total_score })}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                        <button
                                            class="text-accent hover:text-accent-strong"
                                            onclick={(event) => {
                                                event.stopPropagation()
                                                navigate(`/teams/${team.id}`)
                                            }}
                                            type="button"
                                        >
                                            {$t('common.view')}
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                            {#if filteredTeams.length === 0}
                                <tr>
                                    <td colspan="5" class="px-6 py-8 text-center text-sm text-text-muted">
                                        {searchQuery ? $t('teams.noResults') : $t('teams.noTeams')}
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                </div>
            </div>

            {#if filteredTeams.length > 0}
                <p class="mt-4 text-sm text-text-muted">
                    {filteredTeams.length === 1
                        ? $t('teams.countSingular', { count: filteredTeams.length })
                        : $t('teams.countPlural', { count: filteredTeams.length })}
                    {#if searchQuery}
                        {$t('common.outOf', { total: teams.length })}
                    {/if}
                </p>
            {/if}
        </div>
    {/if}
</section>
