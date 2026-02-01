<script lang="ts">
    import { api } from '../../lib/api'
    import { formatApiError, formatDateTime as _formatDateTime, type FieldErrors } from '../../lib/utils'
    import type { RegistrationKey, TeamSummary } from '../../lib/types'
    import { onMount } from 'svelte'

    const formatDateTime = _formatDateTime

    let registrationKeys: RegistrationKey[] = $state([])
    let teams: TeamSummary[] = $state([])
    let keysLoading = $state(false)
    let keysErrorMessage = $state('')
    let teamsLoading = $state(false)
    let teamsErrorMessage = $state('')
    let createKeysLoading = $state(false)
    let createKeysErrorMessage = $state('')
    let createKeysFieldErrors: FieldErrors = $state({})
    let createKeysSuccessMessage = $state('')
    let keyCount = $state(1)
    let selectedTeamId = $state<string>('none')

    onMount(() => {
        loadKeys()
        loadTeams()
    })

    const loadKeys = async () => {
        keysLoading = true
        keysErrorMessage = ''

        try {
            registrationKeys = await api.registrationKeys()
        } catch (error) {
            const formatted = formatApiError(error)
            keysErrorMessage = formatted.message
        } finally {
            keysLoading = false
        }
    }

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

    const submitKeys = async () => {
        createKeysLoading = true
        createKeysErrorMessage = ''
        createKeysSuccessMessage = ''
        createKeysFieldErrors = {}

        try {
            const payload = {
                count: Number(keyCount),
                team_id: selectedTeamId === 'none' ? null : Number(selectedTeamId),
            }
            const created = await api.createRegistrationKeys(payload)
            createKeysSuccessMessage = `${created.length} keys created`
            keyCount = 1
            await loadKeys()
        } catch (error) {
            const formatted = formatApiError(error)
            createKeysErrorMessage = formatted.message
            createKeysFieldErrors = formatted.fieldErrors
        } finally {
            createKeysLoading = false
        }
    }
</script>

<div class="rounded-3xl border border-slate-200 bg-white p-4 dark:border-slate-800/80 dark:bg-slate-900/40 md:p-8">
    <form
        class="space-y-4"
        onsubmit={(event) => {
            event.preventDefault()
            submitKeys()
        }}
    >
        <div class="grid gap-4 md:grid-cols-[1fr_1fr_auto]">
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-key-count"
                    >Count</label
                >
                <input
                    id="admin-key-count"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="number"
                    min="1"
                    bind:value={keyCount}
                />
                {#if createKeysFieldErrors.count}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">count: {createKeysFieldErrors.count}</p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-key-team"
                    >Team</label
                >
                <select
                    id="admin-key-team"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    bind:value={selectedTeamId}
                    disabled={teamsLoading}
                >
                    <option value="none">(not affiliated)</option>
                    {#each teams as team}
                        <option value={team.id}>{team.name}</option>
                    {/each}
                </select>
                {#if createKeysFieldErrors.team_id}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        team_id: {createKeysFieldErrors.team_id}
                    </p>
                {/if}
                {#if teamsErrorMessage}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        {teamsErrorMessage}
                    </p>
                {/if}
            </div>
            <div class="flex items-end">
                <button
                    class="w-full rounded-xl bg-teal-600 px-6 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                    type="submit"
                    disabled={createKeysLoading}
                >
                    {createKeysLoading ? 'Creating...' : 'Create Keys'}
                </button>
            </div>
        </div>

        {#if createKeysErrorMessage}
            <p
                class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
            >
                {createKeysErrorMessage}
            </p>
        {/if}
        {#if createKeysSuccessMessage}
            <p
                class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-700 dark:text-teal-200"
            >
                {createKeysSuccessMessage}
            </p>
        {/if}
    </form>

    <div class="mt-8">
        <div class="flex items-center justify-between">
            <h3 class="text-lg text-slate-900 dark:text-slate-100">Registration Keys</h3>
            <button
                class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
                onclick={loadKeys}
                disabled={keysLoading}
            >
                {keysLoading ? 'Loading...' : 'Refresh'}
            </button>
        </div>

        {#if keysErrorMessage}
            <p
                class="mt-4 rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
            >
                {keysErrorMessage}
            </p>
        {/if}

        {#if keysLoading}
            <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">Loading keys...</p>
        {:else if registrationKeys.length === 0}
            <p class="mt-4 text-sm text-slate-500 dark:text-slate-400">No keys created yet.</p>
        {:else}
            <div class="mt-4 overflow-x-auto">
                <table class="w-full text-left text-sm text-slate-700 dark:text-slate-300">
                    <thead class="text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
                        <tr>
                            <th class="py-2 pr-4">Code</th>
                            <th class="py-2 pr-4">Created by</th>
                            <th class="py-2 pr-4">Team</th>
                            <th class="py-2 pr-4">Created at</th>
                            <th class="py-2 pr-4">Used by</th>
                            <th class="py-2 pr-4">Used IP</th>
                            <th class="py-2">Used at</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each registrationKeys as key}
                            <tr class="border-t border-slate-200/70 dark:border-slate-800/70">
                                <td class="py-3 pr-4 font-mono text-slate-900 dark:text-slate-100">
                                    {key.code}
                                </td>
                                <td class="py-3 pr-4">{key.created_by_username}</td>
                                <td class="py-3 pr-4">{key.team_name ?? '-'}</td>
                                <td class="py-3 pr-4">{formatDateTime(key.created_at)}</td>
                                <td class="py-3 pr-4">{key.used_by_username ?? '-'}</td>
                                <td class="py-3 pr-4 font-mono text-xs">{key.used_by_ip ?? '-'}</td>
                                <td class="py-3">{key.used_at ? formatDateTime(key.used_at) : '-'}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
</div>
