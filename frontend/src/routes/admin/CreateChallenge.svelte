<script lang="ts">
    import { api, uploadPresignedPost } from '../../lib/api'
    import { CHALLENGE_CATEGORIES } from '../../lib/constants'
    import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'
    import FormMessage from '../../components/FormMessage.svelte'

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
                challengeFileError = 'Only .zip files are allowed.'
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

            successMessage = `Challenge "${created.title}" (ID ${created.id}) created successfully`

            if (challengeFile) {
                try {
                    challengeFileUploading = true
                    const uploadResponse = await api.requestChallengeFileUpload(created.id, challengeFile.name)
                    await uploadPresignedPost(uploadResponse.upload, challengeFile)
                    successMessage = `Challenge "${created.title}" (ID ${created.id}) created with file`
                } catch (uploadError) {
                    const formattedUpload = formatApiError(uploadError)
                    errorMessage = `Challenge created, but file upload failed: ${formattedUpload.message}`
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
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-title">Title</label>
            <input
                id="admin-title"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                type="text"
                bind:value={title}
            />
            {#if fieldErrors.title}
                <p class="mt-2 text-xs text-danger">title: {fieldErrors.title}</p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-description">Description</label>
            <textarea
                id="admin-description"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                rows="5"
                bind:value={description}
            ></textarea>
            {#if fieldErrors.description}
                <p class="mt-2 text-xs text-danger">
                    description: {fieldErrors.description}
                </p>
            {/if}
        </div>
        <div class="grid gap-4 md:grid-cols-3">
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-category">Category</label>
                <select
                    id="admin-category"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    bind:value={category}
                >
                    {#each CHALLENGE_CATEGORIES as option}
                        <option value={option}>{option}</option>
                    {/each}
                </select>
                {#if fieldErrors.category}
                    <p class="mt-2 text-xs text-danger">
                        category: {fieldErrors.category}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-points">Points</label>
                <input
                    id="admin-points"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="number"
                    min="1"
                    bind:value={points}
                />
                {#if fieldErrors.points}
                    <p class="mt-2 text-xs text-danger">
                        points: {fieldErrors.points}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-minimum-points">Minimum</label
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
                        minimum_points: {fieldErrors.minimum_points}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-flag">Flag</label>
                <input
                    id="admin-flag"
                    class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                    type="text"
                    bind:value={flag}
                />
                {#if fieldErrors.flag}
                    <p class="mt-2 text-xs text-danger">
                        flag: {fieldErrors.flag}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-file"
                    >Challenge File (.zip)</label
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
            Create as active
        </label>
        <div class="rounded-2xl border border-border bg-surface/60 p-4">
            <label class="flex items-center gap-3 text-sm text-text">
                <input type="checkbox" bind:checked={stackEnabled} class="h-4 w-4 rounded border-border" />
                Provide stack (container instance)
            </label>
            {#if stackEnabled}
                <div class="mt-4 grid gap-4">
                    <div>
                        <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-stack-target-port"
                            >Target Port</label
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
                                stack_target_port: {fieldErrors.stack_target_port}
                            </p>
                        {/if}
                    </div>
                    <div>
                        <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-stack-pod-spec"
                            >Pod Spec (YAML)</label
                        >
                        <textarea
                            id="admin-stack-pod-spec"
                            class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 font-mono text-xs text-text focus:border-accent focus:outline-none"
                            rows="7"
                            bind:value={stackPodSpec}
                        ></textarea>
                        {#if fieldErrors.stack_pod_spec}
                            <p class="mt-2 text-xs text-danger">
                                stack_pod_spec: {fieldErrors.stack_pod_spec}
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
            {loading ? 'Creating...' : challengeFileUploading ? 'Uploading...' : 'Create Challenge'}
        </button>
    </form>
</div>
