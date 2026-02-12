<script lang="ts">
    import { api } from '../../lib/api'
    import { formatApiError, formatDateTime as _formatDateTime, type FieldErrors } from '../../lib/utils'
    import type { RegistrationKey, TeamSummary } from '../../lib/types'
    import { onMount } from 'svelte'
    import FormMessage from '../../components/FormMessage.svelte'

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
    let selectedTeamId = $state<string>('')

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
            if (!selectedTeamId && teams.length > 0) {
                selectedTeamId = String(teams[0].id)
            }
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
            if (!selectedTeamId) {
                createKeysFieldErrors = { team_id: 'required' }
                createKeysLoading = false
                return
            }
            const payload = {
                count: Number(keyCount),
                team_id: Number(selectedTeamId),
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

<div class="rounded-3xl border border-border bg-surface p-4 md:p-8">
    <form
        class="space-y-4"
        onsubmit={(event) => {
            event.preventDefault()
            submitKeys()
        }}
    >
        <div class="grid gap-4 md:grid-cols-[1fr_1fr_auto]">
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-key-count">Count</label>
                <input
                    id="admin-key-count"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="number"
                    min="1"
                    bind:value={keyCount}
                />
                {#if createKeysFieldErrors.count}
                    <p class="mt-2 text-xs text-danger">count: {createKeysFieldErrors.count}</p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-key-team">Team</label>
                <select
                    id="admin-key-team"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    bind:value={selectedTeamId}
                    disabled={teamsLoading}
                >
                    {#each teams as team}
                        <option value={team.id}>{team.name}</option>
                    {/each}
                </select>
                {#if createKeysFieldErrors.team_id}
                    <p class="mt-2 text-xs text-danger">
                        team_id: {createKeysFieldErrors.team_id}
                    </p>
                {/if}
                {#if teamsErrorMessage}
                    <FormMessage variant="error" message={teamsErrorMessage} className="mt-2" />
                {/if}
            </div>
            <div class="flex items-end">
                <button
                    class="w-full rounded-xl bg-accent px-6 py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                    type="submit"
                    disabled={createKeysLoading}
                >
                    {createKeysLoading ? 'Creating...' : 'Create Keys'}
                </button>
            </div>
        </div>

        {#if createKeysErrorMessage}
            <FormMessage variant="error" message={createKeysErrorMessage} />
        {/if}
        {#if createKeysSuccessMessage}
            <FormMessage variant="success" message={createKeysSuccessMessage} />
        {/if}
    </form>

    <div class="mt-8">
        <div class="flex items-center justify-between">
            <h3 class="text-lg text-text">Registration Keys</h3>
            <button
                class="text-xs uppercase tracking-wide text-text-subtle hover:text-text"
                onclick={loadKeys}
                disabled={keysLoading}
            >
                {keysLoading ? 'Loading...' : 'Refresh'}
            </button>
        </div>

        {#if keysErrorMessage}
            <FormMessage variant="error" message={keysErrorMessage} className="mt-4" />
        {/if}

        {#if keysLoading}
            <p class="mt-4 text-sm text-text-subtle">Loading keys...</p>
        {:else if registrationKeys.length === 0}
            <p class="mt-4 text-sm text-text-subtle">No keys created yet.</p>
        {:else}
            <div class="mt-4 overflow-x-auto">
                <table class="w-full text-left text-sm text-text">
                    <thead class="text-xs uppercase tracking-wide text-text-subtle">
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
                            <tr class="border-t border-border/70">
                                <td class="py-3 pr-4 font-mono text-text">
                                    {key.code}
                                </td>
                                <td class="py-3 pr-4">{key.created_by_username}</td>
                                <td class="py-3 pr-4">{key.team_name}</td>
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
