<script lang="ts">
    import { api, uploadPresignedPost } from '../../lib/api'
    import { CHALLENGE_CATEGORIES } from '../../lib/constants'
    import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'
    import FormMessage from '../../components/FormMessage.svelte'
    import { getCategoryKey, t } from '../../lib/i18n'
    import { get } from 'svelte/store'

    let loading = $state(false)
    let errorMessage = $state('')
    let successMessage = $state('')
    let title = $state('')
    let description = $state('')
    let category = $state(CHALLENGE_CATEGORIES[0])
    let points = $state(100)
    let minimumPoints = $state(100)
    let flag = $state('')
    let isActive = $state(true)
    let stackEnabled = $state(false)
    let stackTargetPort = $state(80)
    let stackPodSpec = $state('')
    let challengeFile = $state<File | null>(null)
    let challengeFileError = $state('')
    let challengeFileUploading = $state(false)

    let fieldErrors: FieldErrors = $state({})

    const submit = async () => {
        loading = true
        errorMessage = ''
        successMessage = ''
        fieldErrors = {}
        challengeFileError = ''

        try {
            if (challengeFile && !isZipFile(challengeFile)) {
                challengeFileError = get(t)('admin.create.onlyZip')
                return
            }

            const created = await api.createChallenge({
                title,
                description,
                category,
                points: Number(points),
                minimum_points: Number(minimumPoints),
                flag,
                is_active: isActive,
                stack_enabled: stackEnabled,
                stack_target_port: stackEnabled ? Number(stackTargetPort) : undefined,
                stack_pod_spec: stackEnabled ? stackPodSpec : undefined,
            })

            successMessage = get(t)('admin.create.success', { title: created.title, id: created.id })

            if (challengeFile) {
                try {
                    challengeFileUploading = true
                    const uploadResponse = await api.requestChallengeFileUpload(created.id, challengeFile.name)
                    await uploadPresignedPost(uploadResponse.upload, challengeFile)
                    successMessage = get(t)('admin.create.successWithFile', {
                        title: created.title,
                        id: created.id,
                    })
                } catch (uploadError) {
                    const formattedUpload = formatApiError(uploadError)
                    errorMessage = get(t)('admin.create.fileUploadFailed', { message: formattedUpload.message })
                } finally {
                    challengeFileUploading = false
                }
            }

            title = ''
            description = ''
            category = CHALLENGE_CATEGORIES[0]
            points = 100
            minimumPoints = 100
            flag = ''
            isActive = true
            challengeFile = null
            stackEnabled = false
            stackTargetPort = 80
            stackPodSpec = ''
        } catch (error) {
            const formatted = formatApiError(error)

            errorMessage = formatted.message
            fieldErrors = formatted.fieldErrors
        } finally {
            loading = false
        }
    }
</script>

<div class="rounded-3xl border border-border bg-surface p-4 md:p-8">
    <form
        class="space-y-5"
        onsubmit={(event) => {
            event.preventDefault()
            submit()
        }}
    >
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-title">{$t('common.title')}</label
            >
            <input
                id="admin-title"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                type="text"
                bind:value={title}
            />
            {#if fieldErrors.title}
                <p class="mt-2 text-xs text-danger">{$t('common.title')}: {fieldErrors.title}</p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-description"
                >{$t('common.description')}</label
            >
            <textarea
                id="admin-description"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                rows="5"
                bind:value={description}
            ></textarea>
            {#if fieldErrors.description}
                <p class="mt-2 text-xs text-danger">
                    {$t('common.description')}: {fieldErrors.description}
                </p>
            {/if}
        </div>
        <div class="grid gap-4 md:grid-cols-3">
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-category"
                    >{$t('common.category')}</label
                >
                <select
                    id="admin-category"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    bind:value={category}
                >
                    {#each CHALLENGE_CATEGORIES as option}
                        <option value={option}>{$t(getCategoryKey(option))}</option>
                    {/each}
                </select>
                {#if fieldErrors.category}
                    <p class="mt-2 text-xs text-danger">
                        {$t('common.category')}: {fieldErrors.category}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-points"
                    >{$t('common.points')}</label
                >
                <input
                    id="admin-points"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="number"
                    min="1"
                    bind:value={points}
                />
                {#if fieldErrors.points}
                    <p class="mt-2 text-xs text-danger">
                        {$t('common.points')}: {fieldErrors.points}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-minimum-points"
                    >{$t('common.minimum')}</label
                >
                <input
                    id="admin-minimum-points"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="number"
                    min="0"
                    bind:value={minimumPoints}
                />
                {#if fieldErrors.minimum_points}
                    <p class="mt-2 text-xs text-danger">
                        {$t('common.minimum')}: {fieldErrors.minimum_points}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-flag"
                    >{$t('common.flag')}</label
                >
                <input
                    id="admin-flag"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="text"
                    bind:value={flag}
                />
                {#if fieldErrors.flag}
                    <p class="mt-2 text-xs text-danger">
                        {$t('common.flag')}: {fieldErrors.flag}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-file"
                    >{$t('admin.create.challengeFile')}</label
                >
                <input
                    id="admin-file"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="file"
                    accept=".zip"
                    oninput={(event) => {
                        const target = event.currentTarget as HTMLInputElement
                        challengeFile = target.files?.[0] ?? null
                        challengeFileError = ''
                    }}
                />
                {#if challengeFileError}
                    <p class="mt-2 text-xs text-danger">{challengeFileError}</p>
                {/if}
            </div>
        </div>
        <label class="flex items-center gap-3 text-sm text-text">
            <input type="checkbox" bind:checked={isActive} class="h-4 w-4 rounded border-border" />
            {$t('admin.create.createActive')}
        </label>
        <div class="rounded-2xl border border-border bg-surface/60 p-4">
            <label class="flex items-center gap-3 text-sm text-text">
                <input type="checkbox" bind:checked={stackEnabled} class="h-4 w-4 rounded border-border" />
                {$t('admin.create.provideStack')}
            </label>
            {#if stackEnabled}
                <div class="mt-4 grid gap-4">
                    <div>
                        <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-stack-target-port"
                            >{$t('admin.create.targetPort')}</label
                        >
                        <input
                            id="admin-stack-target-port"
                            class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                            type="number"
                            min="1"
                            max="65535"
                            bind:value={stackTargetPort}
                        />
                        {#if fieldErrors.stack_target_port}
                            <p class="mt-2 text-xs text-danger">
                                {$t('admin.create.targetPort')}: {fieldErrors.stack_target_port}
                            </p>
                        {/if}
                    </div>
                    <div>
                        <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-stack-pod-spec"
                            >{$t('admin.create.podSpec')}</label
                        >
                        <textarea
                            id="admin-stack-pod-spec"
                            class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 font-mono text-xs text-text focus:border-accent focus:outline-none"
                            rows="7"
                            bind:value={stackPodSpec}
                        ></textarea>
                        {#if fieldErrors.stack_pod_spec}
                            <p class="mt-2 text-xs text-danger">
                                {$t('admin.create.podSpec')}: {fieldErrors.stack_pod_spec}
                            </p>
                        {/if}
                    </div>
                </div>
            {/if}
        </div>

        {#if errorMessage}
            <FormMessage variant="error" message={errorMessage} />
        {/if}
        {#if successMessage}
            <FormMessage variant="success" message={successMessage} />
        {/if}

        <button
            class="w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
            type="submit"
            disabled={loading || challengeFileUploading}
        >
            {loading
                ? $t('auth.creating')
                : challengeFileUploading
                  ? $t('admin.create.uploading')
                  : $t('admin.create.createChallenge')}
        </button>
    </form>
</div>
