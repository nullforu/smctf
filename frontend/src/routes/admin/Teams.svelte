<script lang="ts">
    import { api } from '../../lib/api'
    import { formatApiError, formatDateTime as _formatDateTime, type FieldErrors } from '../../lib/utils'
    import type { TeamSummary } from '../../lib/types'
    import { onMount } from 'svelte'
    import FormMessage from '../../components/FormMessage.svelte'
    import { t } from '../../lib/i18n'
    import { get } from 'svelte/store'

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
            createTeamSuccessMessage = get(t)('admin.teams.successCreated', { name: created.name })
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
    <div class="rounded-3xl border border-border bg-surface p-4 md:p-8">
        <form
            class="space-y-4"
            onsubmit={(event) => {
                event.preventDefault()
                submitTeam()
            }}
        >
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-team-name"
                    >{$t('common.teamName')}</label
                >
                <input
                    id="admin-team-name"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="text"
                    bind:value={teamName}
                    placeholder={$t('admin.teams.placeholder')}
                />
                {#if createTeamFieldErrors.name}
                    <p class="mt-2 text-xs text-danger">{$t('common.name')}: {createTeamFieldErrors.name}</p>
                {/if}
            </div>
            {#if createTeamErrorMessage}
                <FormMessage variant="error" message={createTeamErrorMessage} />
            {/if}
            {#if createTeamSuccessMessage}
                <FormMessage variant="success" message={createTeamSuccessMessage} />
            {/if}
            <button
                class="w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                type="submit"
                disabled={createTeamLoading}
            >
                {createTeamLoading ? $t('admin.teams.creating') : $t('admin.teams.createTeam')}
            </button>
        </form>
    </div>

    <div class="rounded-3xl border border-border bg-surface p-4 md:p-8">
        <div class="flex items-center justify-between">
            <h3 class="text-lg text-text">{$t('common.teams')}</h3>
            <button
                class="text-xs uppercase tracking-wide text-text-subtle hover:text-text"
                onclick={loadTeams}
                disabled={teamsLoading}
            >
                {teamsLoading ? $t('common.loading') : $t('common.refresh')}
            </button>
        </div>

        {#if teamsErrorMessage}
            <FormMessage variant="error" message={teamsErrorMessage} className="mt-4" />
        {/if}

        {#if teamsLoading}
            <p class="mt-4 text-sm text-text-subtle">{$t('admin.teams.loadingTeams')}</p>
        {:else if teams.length === 0}
            <p class="mt-4 text-sm text-text-subtle">{$t('admin.teams.noTeams')}</p>
        {:else}
            <div class="mt-4 overflow-x-auto">
                <table class="w-full text-left text-sm text-text">
                    <thead class="text-xs uppercase tracking-wide text-text-subtle">
                        <tr>
                            <th class="py-2 pr-4">{$t('common.id')}</th>
                            <th class="py-2 pr-4">{$t('common.name')}</th>
                            <th class="py-2">{$t('common.createdAt')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each teams as team}
                            <tr class="border-t border-border/70">
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
