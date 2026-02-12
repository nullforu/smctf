<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { TeamSummary } from '../lib/types'
    import { formatApiError } from '../lib/utils'
    import { navigate } from '../lib/router'

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
            <h2 class="text-3xl text-text">Teams</h2>
            <p class="mt-1 text-sm text-text-muted">Browse teams and their stats.</p>
        </div>
    </div>

    <div class="mt-6">
        <input
            type="text"
            placeholder="Search by team name or ID..."
            bind:value={searchQuery}
            class="w-full rounded-xl border border-border bg-surface px-4 py-2.5 text-sm text-text placeholder-text-subtle transition focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent/20"
        />
    </div>

    {#if loading}
        <p class="mt-6 text-sm text-text-muted">Loading...</p>
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
                                    ID
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    Team
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    Members
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    Total Score
                                </th>
                                <th
                                    class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-text-muted"
                                >
                                    Action
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
                                        {team.total_score} pts
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
                                            View
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                            {#if filteredTeams.length === 0}
                                <tr>
                                    <td colspan="5" class="px-6 py-8 text-center text-sm text-text-muted">
                                        {searchQuery ? 'No results found.' : 'No teams found.'}
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                </div>
            </div>

            {#if filteredTeams.length > 0}
                <p class="mt-4 text-sm text-text-muted">
                    {filteredTeams.length} team{filteredTeams.length !== 1 ? 's' : ''}
                    {#if searchQuery}
                        (out of {teams.length})
                    {/if}
                </p>
            {/if}
        </div>
    {/if}
</section>
