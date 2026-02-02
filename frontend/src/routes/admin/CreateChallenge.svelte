<script lang="ts">
    import { api, uploadPresignedPost } from '../../lib/api'
    import { formatApiError, isZipFile, type FieldErrors } from '../../lib/utils'

    const categories = [
        'Web',
        'Web3',
        'Pwnable',
        'Reversing',
        'Crypto',
        'Forensics',
        'Network',
        'Cloud',
        'Misc',
        'Programming',
        'Algorithms',
        'Math',
        'AI',
        'Blockchain',
    ]

    let loading = $state(false)
    let errorMessage = $state('')
    let successMessage = $state('')
    let title = $state('')
    let description = $state('')
    let category = $state(categories[0])
    let points = $state(100)
    let minimumPoints = $state(100)
    let flag = $state('')
    let isActive = $state(true)
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
            category = categories[0]
            points = 100
            minimumPoints = 100
            flag = ''
            isActive = true
            challengeFile = null
        } catch (error) {
            const formatted = formatApiError(error)

            errorMessage = formatted.message
            fieldErrors = formatted.fieldErrors
        } finally {
            loading = false
        }
    }
</script>

<div class="rounded-3xl border border-slate-200 bg-white p-4 dark:border-slate-800/80 dark:bg-slate-900/40 md:p-8">
    <form
        class="space-y-5"
        onsubmit={(event) => {
            event.preventDefault()
            submit()
        }}
    >
        <div>
            <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-title"
                >Title</label
            >
            <input
                id="admin-title"
                class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                type="text"
                bind:value={title}
            />
            {#if fieldErrors.title}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">title: {fieldErrors.title}</p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-description"
                >Description</label
            >
            <textarea
                id="admin-description"
                class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                rows="5"
                bind:value={description}
            ></textarea>
            {#if fieldErrors.description}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                    description: {fieldErrors.description}
                </p>
            {/if}
        </div>
        <div class="grid gap-4 md:grid-cols-3">
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-category"
                    >Category</label
                >
                <select
                    id="admin-category"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    bind:value={category}
                >
                    {#each categories as option}
                        <option value={option}>{option}</option>
                    {/each}
                </select>
                {#if fieldErrors.category}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        category: {fieldErrors.category}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-points"
                    >Points</label
                >
                <input
                    id="admin-points"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="number"
                    min="1"
                    bind:value={points}
                />
                {#if fieldErrors.points}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        points: {fieldErrors.points}
                    </p>
                {/if}
            </div>
            <div>
                <label
                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                    for="admin-minimum-points">Minimum</label
                >
                <input
                    id="admin-minimum-points"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="number"
                    min="0"
                    bind:value={minimumPoints}
                />
                {#if fieldErrors.minimum_points}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        minimum_points: {fieldErrors.minimum_points}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-flag"
                    >Flag</label
                >
                <input
                    id="admin-flag"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="text"
                    bind:value={flag}
                />
                {#if fieldErrors.flag}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                        flag: {fieldErrors.flag}
                    </p>
                {/if}
            </div>
            <div>
                <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-file"
                    >Challenge File (.zip)</label
                >
                <input
                    id="admin-file"
                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                    type="file"
                    accept=".zip"
                    oninput={(event) => {
                        const target = event.currentTarget as HTMLInputElement
                        challengeFile = target.files?.[0] ?? null
                        challengeFileError = ''
                    }}
                />
                {#if challengeFileError}
                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">{challengeFileError}</p>
                {/if}
            </div>
        </div>
        <label class="flex items-center gap-3 text-sm text-slate-700 dark:text-slate-300">
            <input
                type="checkbox"
                bind:checked={isActive}
                class="h-4 w-4 rounded border-slate-300 dark:border-slate-700"
            />
            Create as active
        </label>

        {#if errorMessage}
            <p
                class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
            >
                {errorMessage}
            </p>
        {/if}
        {#if successMessage}
            <p
                class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-700 dark:text-teal-200"
            >
                {successMessage}
            </p>
        {/if}

        <button
            class="w-full rounded-xl bg-teal-600 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
            type="submit"
            disabled={loading || challengeFileUploading}
        >
            {loading ? 'Creating...' : challengeFileUploading ? 'Uploading...' : 'Create Challenge'}
        </button>
    </form>
</div>
