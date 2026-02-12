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
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Teams</h2>
            <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">Browse teams and their stats.</p>
        </div>
    </div>

    <div class="mt-6">
        <input
            type="text"
            placeholder="Search by team name or ID..."
            bind:value={searchQuery}
            class="w-full rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm text-slate-900 placeholder-slate-500 transition focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-500/20 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-100 dark:placeholder-slate-400"
        />
    </div>

    {#if loading}
        <p class="mt-6 text-sm text-slate-600 dark:text-slate-400">Loading...</p>
    {:else if errorMessage}
        <p class="mt-6 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
    {:else}
        <div class="mt-6">
            <div
                class="overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-slate-800/80 dark:bg-slate-900/40"
            >
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead class="border-b border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/60">
                            <tr>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    ID
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Team
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Members
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Total Score
                                </th>
                                <th
                                    class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Action
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-200 dark:divide-slate-800">
                            {#each filteredTeams as team}
                                <tr
                                    class="transition hover:bg-slate-50 dark:hover:bg-slate-900/60 cursor-pointer"
                                    onclick={() => navigate(`/teams/${team.id}`)}
                                >
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                        {team.id}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                        {team.name}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                        {team.member_count}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-teal-600 dark:text-teal-200">
                                        {team.total_score} pts
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                        <button
                                            class="text-teal-600 hover:text-teal-700 dark:text-teal-400 dark:hover:text-teal-300"
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
                                    <td
                                        colspan="5"
                                        class="px-6 py-8 text-center text-sm text-slate-600 dark:text-slate-400"
                                    >
                                        {searchQuery ? 'No results found.' : 'No teams found.'}
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                </div>
            </div>

            {#if filteredTeams.length > 0}
                <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">
                    {filteredTeams.length} team{filteredTeams.length !== 1 ? 's' : ''}
                    {#if searchQuery}
                        (out of {teams.length})
                    {/if}
                </p>
            {/if}
        </div>
    {/if}
</section>
