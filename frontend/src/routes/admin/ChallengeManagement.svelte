<script lang="ts">
    import { api } from '../../lib/api'
    import { formatApiError, type FieldErrors } from '../../lib/utils'
    import type { Challenge } from '../../lib/types'
    import { onMount } from 'svelte'

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

    let challenges: Challenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let successMessage = $state('')
    let expandedChallengeId: number | null = $state(null)
    let manageLoading = $state(false)
    let manageFieldErrors: FieldErrors = $state({})
    let editTitle = $state('')
    let editDescription = $state('')
    let editCategory = $state(categories[0])
    let editPoints = $state(100)
    let editMinimumPoints = $state(100)
    let editIsActive = $state(true)

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

    const openEditor = (challenge: Challenge) => {
        manageFieldErrors = {}
        errorMessage = ''
        successMessage = ''

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
            })

            challenges = challenges.map((item) => (item.id === updated.id ? updated : item))
            successMessage = `Challenge "${updated.title}" updated successfully`

            editTitle = updated.title
            editDescription = updated.description
            editCategory = updated.category
            editPoints = updated.initial_points
            editMinimumPoints = updated.minimum_points
            editIsActive = updated.is_active
        } catch (error) {
            const formatted = formatApiError(error)
            errorMessage = formatted.message
            manageFieldErrors = formatted.fieldErrors
        } finally {
            manageLoading = false
        }
    }

    const deleteChallenge = async (challenge: Challenge) => {
        const confirmed = window.confirm(`Delete challenge "${challenge.title}" (ID ${challenge.id})?`)
        if (!confirmed) return

        manageLoading = true
        manageFieldErrors = {}
        errorMessage = ''
        successMessage = ''

        try {
            await api.deleteChallenge(challenge.id)
            challenges = challenges.filter((item) => item.id !== challenge.id)
            successMessage = `Challenge "${challenge.title}" deleted`
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
            class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            onclick={loadChallenges}
            disabled={loading}
        >
            {loading ? 'Loading...' : 'Refresh'}
        </button>
    </div>

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

    {#if loading}
        <p class="text-sm text-slate-500 dark:text-slate-400">Loading challenges...</p>
    {:else}
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
                                Title
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Category
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Points
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Initial
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Minimum
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Solved
                            </th>
                            <th
                                class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Status
                            </th>
                            <th
                                class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                            >
                                Action
                            </th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-slate-200 dark:divide-slate-800">
                        {#each challenges as challenge (challenge.id)}
                            <tr class="transition hover:bg-slate-50 dark:hover:bg-slate-900/60">
                                <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                    {challenge.id}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                    {challenge.title}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                    {challenge.category}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                    {challenge.points}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                    {challenge.initial_points}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                    {challenge.minimum_points}
                                </td>
                                <td class="px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                    {challenge.solve_count}
                                </td>
                                <td class="px-6 py-4 text-sm">
                                    <span
                                        class={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium uppercase ${
                                            challenge.is_active
                                                ? 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300'
                                                : 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
                                        }`}
                                    >
                                        {challenge.is_active ? 'active' : 'inactive'}
                                    </span>
                                </td>
                                <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                    <div class="flex items-center justify-end gap-3">
                                        <button
                                            class="text-teal-600 hover:text-teal-700 dark:text-teal-400 dark:hover:text-teal-300"
                                            onclick={() => openEditor(challenge)}
                                            disabled={manageLoading}
                                        >
                                            {expandedChallengeId === challenge.id ? 'Close Edit' : 'Edit'}
                                        </button>
                                        <button
                                            class="text-rose-600 hover:text-rose-700 dark:text-rose-400 dark:hover:text-rose-300"
                                            onclick={() => deleteChallenge(challenge)}
                                            disabled={manageLoading}
                                        >
                                            Delete
                                        </button>
                                    </div>
                                </td>
                            </tr>
                            {#if expandedChallengeId === challenge.id}
                                <tr class="bg-slate-50/70 dark:bg-slate-900/40">
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
                                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                    for={`manage-title-${challenge.id}`}>Title</label
                                                >
                                                <input
                                                    id={`manage-title-${challenge.id}`}
                                                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                    type="text"
                                                    bind:value={editTitle}
                                                />
                                                {#if manageFieldErrors.title}
                                                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                                        title: {manageFieldErrors.title}
                                                    </p>
                                                {/if}
                                            </div>
                                            <div>
                                                <label
                                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                    for={`manage-description-${challenge.id}`}>Description</label
                                                >
                                                <textarea
                                                    id={`manage-description-${challenge.id}`}
                                                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                    rows="5"
                                                    bind:value={editDescription}
                                                ></textarea>
                                                {#if manageFieldErrors.description}
                                                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                                        description: {manageFieldErrors.description}
                                                    </p>
                                                {/if}
                                            </div>
                                            <div class="grid gap-4 md:grid-cols-3">
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                        for={`manage-category-${challenge.id}`}>Category</label
                                                    >
                                                    <select
                                                        id={`manage-category-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                        bind:value={editCategory}
                                                    >
                                                        {#each categories as option}
                                                            <option value={option}>{option}</option>
                                                        {/each}
                                                    </select>
                                                    {#if manageFieldErrors.category}
                                                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                                            category: {manageFieldErrors.category}
                                                        </p>
                                                    {/if}
                                                </div>
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                        for={`manage-points-${challenge.id}`}>Points</label
                                                    >
                                                    <input
                                                        id={`manage-points-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                        type="number"
                                                        min="1"
                                                        bind:value={editPoints}
                                                    />
                                                    {#if manageFieldErrors.points}
                                                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                                            points: {manageFieldErrors.points}
                                                        </p>
                                                    {/if}
                                                </div>
                                                <div>
                                                    <label
                                                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                        for={`manage-minimum-points-${challenge.id}`}>Minimum</label
                                                    >
                                                    <input
                                                        id={`manage-minimum-points-${challenge.id}`}
                                                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                        type="number"
                                                        min="0"
                                                        bind:value={editMinimumPoints}
                                                    />
                                                    {#if manageFieldErrors.minimum_points}
                                                        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                                            minimum_points: {manageFieldErrors.minimum_points}
                                                        </p>
                                                    {/if}
                                                </div>
                                            </div>
                                            <label
                                                class="flex items-center gap-3 text-sm text-slate-700 dark:text-slate-300"
                                            >
                                                <input
                                                    type="checkbox"
                                                    bind:checked={editIsActive}
                                                    class="h-4 w-4 rounded border-slate-300 dark:border-slate-700"
                                                />
                                                Active
                                            </label>

                                            <div class="flex flex-col gap-3 sm:flex-row sm:justify-end">
                                                <button
                                                    class="rounded-xl border border-slate-300 px-5 py-3 text-sm text-slate-700 transition hover:border-slate-400 hover:text-slate-900 disabled:opacity-60 dark:border-slate-700 dark:text-slate-200 dark:hover:border-slate-500"
                                                    type="button"
                                                    onclick={() => (expandedChallengeId = null)}
                                                    disabled={manageLoading}
                                                >
                                                    Cancel
                                                </button>
                                                <button
                                                    class="rounded-xl bg-teal-600 px-5 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
                                                    type="submit"
                                                    disabled={manageLoading}
                                                >
                                                    {manageLoading ? 'Saving...' : 'Save Changes'}
                                                </button>
                                            </div>
                                        </form>
                                    </td>
                                </tr>
                            {/if}
                        {/each}
                        {#if challenges.length === 0}
                            <tr>
                                <td
                                    colspan="9"
                                    class="px-6 py-8 text-center text-sm text-slate-600 dark:text-slate-400"
                                >
                                    No challenges found.
                                </td>
                            </tr>
                        {/if}
                    </tbody>
                </table>
            </div>
        </div>
    {/if}
</div>
