<script lang="ts">
    import { api, uploadPresignedPost } from '../../lib/api'
    import { CHALLENGE_CATEGORIES } from '../../lib/constants'
    import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'
    import type { Challenge } from '../../lib/types'
    import { onMount } from 'svelte'
    import FormMessage from '../../components/FormMessage.svelte'
    import { get } from 'svelte/store'
    import { getCategoryKey, t } from '../../lib/i18n'

    let challenges: Challenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let successMessage = $state('')
    let expandedChallengeId: number | null = $state(null)
    let manageLoading = $state(false)
    let manageFieldErrors: FieldErrors = $state({})
    let editTitle = $state('')
    let editDescription = $state('')
    let editCategory = $state<string>(CHALLENGE_CATEGORIES[0])
    let editPoints = $state(100)
    let editMinimumPoints = $state(100)
    let editIsActive = $state(true)
    let editStackEnabled = $state(false)
    let editStackTargetPort = $state(80)
    let editStackPodSpec = $state('')
    let editFile = $state<File | null>(null)
    let editFileError = $state('')
    let editFileUploading = $state(false)
    let editFileSuccess = $state('')

    onMount(() => {
        loadChallenges()
    })

    const loadChallenges = async () => {
        loading = true
        errorMessage = ''

        try {
            challenges = await api.challenges()
        } catch (error) {
            const formatted = formatApiError(error)
            errorMessage = formatted.message
        } finally {
            loading = false
        }
    }

    const openEditor = async (challenge: Challenge) => {
        manageFieldErrors = {}
        errorMessage = ''
        successMessage = ''
        editFileError = ''
        editFileSuccess = ''
        editFile = null

        if (expandedChallengeId === challenge.id) {
            expandedChallengeId = null
            return
        }

        expandedChallengeId = challenge.id
        editTitle = challenge.title
        editDescription = challenge.description
        editCategory = challenge.category
        editPoints = challenge.initial_points
        editMinimumPoints = challenge.minimum_points
        editIsActive = challenge.is_active
        editStackEnabled = challenge.stack_enabled
        editStackTargetPort = challenge.stack_target_port || 80
        editStackPodSpec = ''

        if (editStackEnabled) {
            try {
                manageLoading = true
                const detail = await api.adminChallenge(challenge.id)
                editStackTargetPort = detail.stack_target_port || editStackTargetPort
                editStackPodSpec = detail.stack_pod_spec ?? ''
            } catch (error) {
                const formatted = formatApiError(error)
                errorMessage = formatted.message
            } finally {
                manageLoading = false
            }
        }
    }

    const submitEdit = async (challenge: Challenge) => {
        manageLoading = true
        manageFieldErrors = {}
        errorMessage = ''
        successMessage = ''

        try {
            const updated = await api.updateChallenge(challenge.id, {
                title: editTitle,
                description: editDescription,
                category: editCategory,
                points: Number(editPoints),
                minimum_points: Number(editMinimumPoints),
                is_active: editIsActive,
                stack_enabled: editStackEnabled,
                stack_target_port: editStackEnabled ? Number(editStackTargetPort) : undefined,
                stack_pod_spec: editStackEnabled && editStackPodSpec.trim() ? editStackPodSpec : undefined,
            })

            challenges = challenges.map((item) => (item.id === updated.id ? updated : item))
            successMessage = get(t)('admin.manage.successUpdated', { title: updated.title })

            editTitle = updated.title
            editDescription = updated.description
            editCategory = updated.category
            editPoints = updated.initial_points
            editMinimumPoints = updated.minimum_points
            editIsActive = updated.is_active
            editStackEnabled = updated.stack_enabled
            editStackTargetPort = updated.stack_target_port || 80
            editStackPodSpec = ''
        } catch (error) {
            const formatted = formatApiError(error)
            errorMessage = formatted.message
            manageFieldErrors = formatted.fieldErrors
        } finally {
            manageLoading = false
        }
    }

    const uploadEditFile = async (challenge: Challenge) => {
        editFileError = ''
        editFileSuccess = ''

        if (!editFile) {
            editFileError = get(t)('admin.manage.selectZip')
            return
        }

        if (!isZipFile(editFile)) {
            editFileError = get(t)('admin.create.onlyZip')
            return
        }

        editFileUploading = true

        try {
            const uploadResponse = await api.requestChallengeFileUpload(challenge.id, editFile.name)
            await uploadPresignedPost(uploadResponse.upload, editFile)
            challenges = challenges.map((item) =>
                item.id === uploadResponse.challenge.id ? uploadResponse.challenge : item,
            )
            editFileSuccess = get(t)('admin.manage.fileUploaded')
            editFile = null
        } catch (error) {
            const formatted = formatApiError(error)
            editFileError = formatted.message
        } finally {
            editFileUploading = false
        }
    }

    const deleteEditFile = async (challenge: Challenge) => {
        const confirmed = window.confirm(
            get(t)('admin.manage.confirmDeleteFile', { title: challenge.title, id: challenge.id }),
        )
        if (!confirmed) return

        editFileError = ''
        editFileSuccess = ''
        editFileUploading = true

        try {
            const updated = await api.deleteChallengeFile(challenge.id)
            challenges = challenges.map((item) => (item.id === updated.id ? updated : item))
            editFileSuccess = get(t)('admin.manage.fileDeleted')
        } catch (error) {
            const formatted = formatApiError(error)
            editFileError = formatted.message
        } finally {
            editFileUploading = false
        }
    }

    const deleteChallenge = async (challenge: Challenge) => {
        const confirmed = window.confirm(
            get(t)('admin.manage.confirmDeleteChallenge', { title: challenge.title, id: challenge.id }),
        )
        if (!confirmed) return

        manageLoading = true
        manageFieldErrors = {}
        errorMessage = ''
        successMessage = ''

        try {
            await api.deleteChallenge(challenge.id)
            challenges = challenges.filter((item) => item.id !== challenge.id)
            successMessage = get(t)('admin.manage.successDeleted', { title: challenge.title })
            if (expandedChallengeId === challenge.id) {
                expandedChallengeId = null
            }
        } catch (error) {
            const formatted = formatApiError(error)
            errorMessage = formatted.message
        } finally {
            manageLoading = false
        }
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <button
            class="text-xs uppercase tracking-wide text-text-subtle hover:text-text"
            onclick={loadChallenges}
            disabled={loading}
        >
            {loading ? $t('common.loading') : $t('common.refresh')}
        </button>
    </div>

    {#if errorMessage}
        <FormMessage variant="error" message={errorMessage} />
    {/if}
    {#if successMessage}
        <FormMessage variant="success" message={successMessage} />
    {/if}

    {#if loading}
        <p class="text-sm text-text-subtle">{$t('admin.manage.loadingChallenges')}</p>
    {:else}
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
                                {$t('common.title')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('common.category')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('admin.manage.initial')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('common.points')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('common.minimum')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('challenges.solvedLabel')}
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('common.status')}
                            </th>
                            <th
                                class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-text-muted"
                            >
                                {$t('common.action')}
                            </th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-border">
                        {#each challenges as challenge (challenge.id)}
                            <tr class="transition hover:bg-surface-muted">
                                <td class="whitespace-nowrap px-6 py-4 text-sm text-text">
                                    {challenge.id}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {challenge.title}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {$t(getCategoryKey(challenge.category))}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {challenge.points}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {challenge.initial_points}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {challenge.minimum_points}
                                </td>
                                <td class="px-6 py-4 text-sm text-text">
                                    {challenge.solve_count}
                                </td>
                                <td class="px-6 py-4 text-sm">
                                    <span
                                        class={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase ${
                                            challenge.is_active
                                                ? 'bg-accent/20 text-accent-strong  '
                                                : 'bg-surface-subtle text-text  '
                                        }`}
                                    >
                                        {challenge.is_active
                                            ? $t('admin.manage.statusActive')
                                            : $t('admin.manage.statusInactive')}
                                    </span>
                                </td>
                                <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                    <div class="flex items-center justify-end gap-3">
                                        <button
                                            class="text-accent hover:text-accent-strong"
                                            onclick={() => openEditor(challenge)}
                                            disabled={manageLoading}
                                        >
                                            {expandedChallengeId === challenge.id
                                                ? $t('admin.manage.closeEdit')
                                                : $t('admin.manage.edit')}
                                        </button>
                                        <button
                                            class="text-danger hover:text-danger-strong"
                                            onclick={() => deleteChallenge(challenge)}
                                            disabled={manageLoading}
                                        >
                                            {$t('admin.manage.delete')}
                                        </button>
                                    </div>
                                </td>
                            </tr>
                            {#if expandedChallengeId === challenge.id}
                                <tr class="bg-surface/70">
                                    <td colspan="9" class="px-6 py-6">
                                        <form
                                            class="space-y-5"
                                            onsubmit={(event) => {
                                                event.preventDefault()
                                                submitEdit(challenge)
                                            }}
                                        >
                                            <div>
                                                <label
                                                    class="text-xs uppercase tracking-wide text-text-muted"
                                                    for={`manage-title-${challenge.id}`}>{$t('common.title')}</label
                                                >
                                                <input
                                                    id={`manage-title-${challenge.id}`}
                                                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                    type="text"
                                                    bind:value={editTitle}
                                                />
                                                {#if manageFieldErrors.title}
                                                    <p class="mt-2 text-xs text-danger">
                                                        {$t('common.title')}: {manageFieldErrors.title}
                                                    </p>
                                                {/if}
                                            </div>
                                            <div>
                                                <label
                                                    class="text-xs uppercase tracking-wide text-text-muted"
                                                    for={`manage-description-${challenge.id}`}
                                                    >{$t('common.description')}</label
                                                >
                                                <textarea
                                                    id={`manage-description-${challenge.id}`}
                                                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                    rows="5"
                                                    bind:value={editDescription}
                                                ></textarea>
                                                {#if manageFieldErrors.description}
                                                    <p class="mt-2 text-xs text-danger">
                                                        {$t('common.description')}: {manageFieldErrors.description}
                                                    </p>
                                                {/if}
                                            </div>
                                            <div class="grid gap-4 md:grid-cols-3">
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-text-muted"
                                                        for={`manage-category-${challenge.id}`}
                                                        >{$t('common.category')}</label
                                                    >
                                                    <select
                                                        id={`manage-category-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                        bind:value={editCategory}
                                                    >
                                                        {#each CHALLENGE_CATEGORIES as option}
                                                            <option value={option}>{$t(getCategoryKey(option))}</option>
                                                        {/each}
                                                    </select>
                                                    {#if manageFieldErrors.category}
                                                        <p class="mt-2 text-xs text-danger">
                                                            {$t('common.category')}: {manageFieldErrors.category}
                                                        </p>
                                                    {/if}
                                                </div>
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-text-muted"
                                                        for={`manage-points-${challenge.id}`}
                                                        >{$t('common.points')}</label
                                                    >
                                                    <input
                                                        id={`manage-points-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                        type="number"
                                                        min="1"
                                                        bind:value={editPoints}
                                                    />
                                                    {#if manageFieldErrors.points}
                                                        <p class="mt-2 text-xs text-danger">
                                                            {$t('common.points')}: {manageFieldErrors.points}
                                                        </p>
                                                    {/if}
                                                </div>
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-text-muted"
                                                        for={`manage-minimum-points-${challenge.id}`}
                                                        >{$t('common.minimum')}</label
                                                    >
                                                    <input
                                                        id={`manage-minimum-points-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                        type="number"
                                                        min="0"
                                                        bind:value={editMinimumPoints}
                                                    />
                                                    {#if manageFieldErrors.minimum_points}
                                                        <p class="mt-2 text-xs text-danger">
                                                            {$t('common.minimum')}: {manageFieldErrors.minimum_points}
                                                        </p>
                                                    {/if}
                                                </div>
                                            </div>
                                            <label class="flex items-center gap-3 text-sm text-text">
                                                <input
                                                    type="checkbox"
                                                    bind:checked={editIsActive}
                                                    class="h-4 w-4 rounded border-border"
                                                />
                                                {$t('common.active')}
                                            </label>
                                            <div class="rounded-2xl border border-border bg-surface/60 p-4">
                                                <label class="flex items-center gap-3 text-sm text-text">
                                                    <input
                                                        type="checkbox"
                                                        bind:checked={editStackEnabled}
                                                        class="h-4 w-4 rounded border-border"
                                                    />
                                                    {$t('admin.create.provideStack')}
                                                </label>
                                                {#if editStackEnabled}
                                                    <div class="mt-4 grid gap-4">
                                                        <div>
                                                            <label
                                                                class="text-xs uppercase tracking-wide text-text-muted"
                                                                for={`manage-stack-target-port-${challenge.id}`}
                                                                >{$t('admin.create.targetPort')}</label
                                                            >
                                                            <input
                                                                id={`manage-stack-target-port-${challenge.id}`}
                                                                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                                                                type="number"
                                                                min="1"
                                                                max="65535"
                                                                bind:value={editStackTargetPort}
                                                            />
                                                            {#if manageFieldErrors.stack_target_port}
                                                                <p class="mt-2 text-xs text-danger">
                                                                    {$t('admin.create.targetPort')}: {manageFieldErrors.stack_target_port}
                                                                </p>
                                                            {/if}
                                                        </div>
                                                        <div>
                                                            <label
                                                                class="text-xs uppercase tracking-wide text-text-muted"
                                                                for={`manage-stack-pod-spec-${challenge.id}`}
                                                                >{$t('admin.create.podSpec')}</label
                                                            >
                                                            <textarea
                                                                id={`manage-stack-pod-spec-${challenge.id}`}
                                                                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 font-mono text-xs text-text focus:border-accent focus:outline-none"
                                                                rows="7"
                                                                placeholder={$t('admin.manage.podSpecPlaceholder')}
                                                                bind:value={editStackPodSpec}
                                                            ></textarea>
                                                            {#if manageFieldErrors.stack_pod_spec}
                                                                <p class="mt-2 text-xs text-danger">
                                                                    {$t('admin.create.podSpec')}: {manageFieldErrors.stack_pod_spec}
                                                                </p>
                                                            {/if}
                                                        </div>
                                                    </div>
                                                {/if}
                                            </div>

                                            <div
                                                class="rounded-xl border border-border bg-surface/60 p-4 text-sm text-text"
                                            >
                                                <p class="text-xs uppercase tracking-wide text-text-subtle">
                                                    {$t('admin.manage.challengeFile')}
                                                </p>
                                                <p class="mt-2 text-sm text-text">
                                                    {challenge.has_file
                                                        ? (challenge.file_name ?? 'challenge.zip')
                                                        : $t('admin.manage.noFileUploaded')}
                                                </p>
                                                <div class="mt-3 flex flex-wrap items-center gap-3">
                                                    <input
                                                        class="w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-text sm:w-auto"
                                                        type="file"
                                                        accept=".zip"
                                                        oninput={(event) => {
                                                            const target = event.currentTarget as HTMLInputElement
                                                            editFile = target.files?.[0] ?? null
                                                            editFileError = ''
                                                            editFileSuccess = ''
                                                        }}
                                                    />
                                                    <button
                                                        class="rounded-lg bg-contrast px-4 py-2 text-xs font-medium text-contrast-foreground transition hover:bg-contrast/80 disabled:opacity-60"
                                                        type="button"
                                                        onclick={() => uploadEditFile(challenge)}
                                                        disabled={editFileUploading || manageLoading}
                                                    >
                                                        {editFileUploading
                                                            ? $t('admin.create.uploading')
                                                            : $t('admin.manage.uploadZip')}
                                                    </button>
                                                    {#if challenge.has_file}
                                                        <button
                                                            class="rounded-lg border border-danger/30 px-4 py-2 text-xs font-medium text-danger transition hover:border-danger/50 hover:text-danger-strong disabled:opacity-60"
                                                            type="button"
                                                            onclick={() => deleteEditFile(challenge)}
                                                            disabled={editFileUploading || manageLoading}
                                                        >
                                                            {$t('admin.manage.deleteFile')}
                                                        </button>
                                                    {/if}
                                                </div>
                                                {#if editFileError}
                                                    <FormMessage
                                                        variant="error"
                                                        message={editFileError}
                                                        className="mt-2"
                                                    />
                                                {/if}
                                                {#if editFileSuccess}
                                                    <FormMessage
                                                        variant="success"
                                                        message={editFileSuccess}
                                                        className="mt-2"
                                                    />
                                                {/if}
                                            </div>

                                            <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
                                                <button
                                                    class="rounded-xl border border-border px-5 py-3 text-sm text-text transition hover:border-border hover:text-text disabled:opacity-60"
                                                    type="button"
                                                    onclick={() => (expandedChallengeId = null)}
                                                    disabled={manageLoading}
                                                >
                                                    {$t('common.cancel')}
                                                </button>
                                                <button
                                                    class="rounded-xl bg-accent px-5 py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
                                                    type="submit"
                                                    disabled={manageLoading}
                                                >
                                                    {manageLoading ? $t('admin.site.saving') : $t('common.saveChanges')}
                                                </button>
                                            </div>
                                        </form>
                                    </td>
                                </tr>
                            {/if}
                        {/each}
                        {#if challenges.length === 0}
                            <tr>
                                <td colspan="9" class="px-6 py-8 text-center text-sm text-text-muted">
                                    {$t('admin.manage.noChallenges')}
                                </td>
                            </tr>
                        {/if}
                    </tbody>
                </table>
            </div>
        </div>
    {/if}
</div>
