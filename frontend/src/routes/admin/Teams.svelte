<script lang="ts">
    import { api } from '../../lib/api'
    import { formatApiError, formatDateTime as _formatDateTime, type FieldErrors } from '../../lib/utils'
    import type { TeamSummary } from '../../lib/types'
    import { onMount } from 'svelte'

    const formatDateTime = _formatDateTime

    let teams: TeamSummary[] = $state([])
    let teamsLoading = $state(false)
    let teamsErrorMessage = $state('')
    let teamName = $state('')
    let createTeamLoading = $state(false)
    let createTeamErrorMessage = $state('')
    let createTeamSuccessMessage = $state('')
    let createTeamFieldErrors: FieldErrors = $state({})

    onMount(() => {
        loadTeams()
    })

    const loadTeams = async () => {
        teamsLoading = true
        teamsErrorMessage = ''

        try {
            teams = await api.teams()
        } catch (error) {
            const formatted = formatApiError(error)
            teamsErrorMessage = formatted.message
        } finally {
            teamsLoading = false
        }
    }

    const submitTeam = async () => {
        createTeamLoading = true
        createTeamErrorMessage = ''
        createTeamSuccessMessage = ''
        createTeamFieldErrors = {}

        try {
            const created = await api.createTeam({ name: teamName })
            createTeamSuccessMessage = `Team "${created.name}" created`
            teamName = ''
            await loadTeams()
        } catch (error) {
            const formatted = formatApiError(error)
            createTeamErrorMessage = formatted.message
            createTeamFieldErrors = formatted.fieldErrors
        } finally {
            createTeamLoading = false
        }
    }
</script>

<div class="space-y-6">
    <div class="rounded-3xl border border-slate-200 bg-white p-4 dark:border-slate-800/80 dark:bg-slate-900/40 md:p-8">
        <form
            class="space-y-4"
            onsubmit={(event) => {
                event.preventDefault()
                submitTeam()
            }}
        >
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-team-name"
                    >Team Name</label
                >
                <input
                    id="admin-team-name"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="text"
                    bind:value={teamName}
                    placeholder="e.g., 세명컴퓨터고등학교 or Null4U"
                />
                {#if createTeamFieldErrors.name}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">name: {createTeamFieldErrors.name}</p>
                {/if}
            </div>
            {#if createTeamErrorMessage}
                <p
                    class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
                >
                    {createTeamErrorMessage}
                </p>
            {/if}
            {#if createTeamSuccessMessage}
                <p
                    class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-700 dark:text-teal-200"
                >
                    {createTeamSuccessMessage}
                </p>
            {/if}
            <button
                class="w-full rounded-xl bg-teal-600 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                type="submit"
                disabled={createTeamLoading}
            >
                {createTeamLoading ? 'Creating...' : 'Create Team'}
            </button>
        </form>
    </div>

    <div class="rounded-3xl border border-slate-200 bg-white p-4 dark:border-slate-800/80 dark:bg-slate-900/40 md:p-8">
        <div class="flex items-center justify-between">
            <h3 class="text-lg text-slate-900 dark:text-slate-100">Teams</h3>
            <button
                class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
                onclick={loadTeams}
                disabled={teamsLoading}
            >
                {teamsLoading ? 'Loading...' : 'Refresh'}
            </button>
        </div>

        {#if teamsErrorMessage}
            <p
                class="mt-4 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
            >
                {teamsErrorMessage}
            </p>
        {/if}

        {#if teamsLoading}
            <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">Loading teams...</p>
        {:else if teams.length === 0}
            <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">No teams created yet.</p>
        {:else}
            <div class="mt-4 overflow-x-auto">
                <table class="w-full text-left text-sm text-slate-700 dark:text-slate-300">
                    <thead class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4">ID</th>
                            <th class="py-2 pr-4">Name</th>
                            <th class="py-2">Created at</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each teams as team}
                            <tr class="border-t border-slate-200/70 dark:border-slate-800/70">
                                <td class="py-3 pr-4">{team.id}</td>
                                <td class="py-3 pr-4">{team.name}</td>
                                <td class="py-3">{formatDateTime(team.created_at)}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
</div>
