<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api } from '../lib/api'
    import type { Challenge, RegistrationKey } from '../lib/types'
    import { formatApiError, formatDateTime as _formatDateTime, type FieldErrors } from '../lib/utils'

    const formatDateTime = _formatDateTime

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let activeTab = $state<'challenges' | 'challenge_management' | 'registration_keys'>('challenges')
    let title = $state('')
    let description = $state('')
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
    let category = $state(categories[0])
    let points = $state(100)
    let flag = $state('')
    let isActive = $state(true)

    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let successMessage = $state('')
    let registrationKeys: RegistrationKey[] = $state([])
    let keysLoading = $state(false)
    let keysErrorMessage = $state('')
    let createKeysLoading = $state(false)
    let createKeysErrorMessage = $state('')
    let createKeysFieldErrors: FieldErrors = $state({})
    let createKeysSuccessMessage = $state('')
    let keyCount = $state(1)
    let challenges: Challenge[] = $state([])
    let challengesLoading = $state(false)
    let challengesErrorMessage = $state('')
    let challengesSuccessMessage = $state('')
    let challengesLoaded = $state(false)
    let expandedChallengeId: number | null = $state(null)
    let manageLoading = $state(false)
    let manageFieldErrors: FieldErrors = $state({})
    let editTitle = $state('')
    let editDescription = $state('')
    let editCategory = $state(categories[0])
    let editPoints = $state(100)
    let editIsActive = $state(true)
    let auth = $state(get(authStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    $effect(() => {
        if (auth.user?.role === 'admin' && activeTab === 'registration_keys') {
            loadKeys()
        }
    })

    $effect(() => {
        if (auth.user?.role === 'admin' && activeTab === 'challenge_management' && !challengesLoaded) {
            loadChallenges()
        }
    })

    const submit = async () => {
        loading = true
        errorMessage = ''
        successMessage = ''
        fieldErrors = {}

        try {
            const created = await api.createChallenge({
                title,
                description,
                category,
                points: Number(points),
                flag,
                is_active: isActive,
            })

            successMessage = `Challenge "${created.title}" (ID ${created.id}) created successfully`
            title = ''
            description = ''
            category = categories[0]
            points = 100
            flag = ''
            isActive = true
            challengesLoaded = false
        } catch (error) {
            const formatted = formatApiError(error)

            errorMessage = formatted.message
            fieldErrors = formatted.fieldErrors
        } finally {
            loading = false
        }
    }

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

    const loadChallenges = async () => {
        challengesLoading = true
        challengesErrorMessage = ''

        try {
            challenges = await api.challenges()
            challengesLoaded = true
        } catch (error) {
            const formatted = formatApiError(error)
            challengesErrorMessage = formatted.message
        } finally {
            challengesLoading = false
        }
    }

    const openEditor = (challenge: Challenge) => {
        manageFieldErrors = {}
        challengesErrorMessage = ''
        challengesSuccessMessage = ''

        if (expandedChallengeId === challenge.id) {
            expandedChallengeId = null
            return
        }

        expandedChallengeId = challenge.id
        editTitle = challenge.title
        editDescription = challenge.description
        editCategory = challenge.category
        editPoints = challenge.points
        editIsActive = challenge.is_active
    }

    const submitEdit = async (challenge: Challenge) => {
        manageLoading = true
        manageFieldErrors = {}
        challengesErrorMessage = ''
        challengesSuccessMessage = ''

        try {
            const updated = await api.updateChallenge(challenge.id, {
                title: editTitle,
                description: editDescription,
                category: editCategory,
                points: Number(editPoints),
                is_active: editIsActive,
            })

            challenges = challenges.map((item) => (item.id === updated.id ? updated : item))
            challengesSuccessMessage = `Challenge "${updated.title}" updated successfully`

            editTitle = updated.title
            editDescription = updated.description
            editCategory = updated.category
            editPoints = updated.points
            editIsActive = updated.is_active
        } catch (error) {
            const formatted = formatApiError(error)
            challengesErrorMessage = formatted.message
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
        challengesErrorMessage = ''
        challengesSuccessMessage = ''

        try {
            await api.deleteChallenge(challenge.id)
            challenges = challenges.filter((item) => item.id !== challenge.id)
            challengesSuccessMessage = `Challenge "${challenge.title}" deleted`
            if (expandedChallengeId === challenge.id) {
                expandedChallengeId = null
            }
        } catch (error) {
            const formatted = formatApiError(error)
            challengesErrorMessage = formatted.message
        } finally {
            manageLoading = false
        }
    }

    const submitKeys = async () => {
        createKeysLoading = true
        createKeysErrorMessage = ''
        createKeysSuccessMessage = ''
        createKeysFieldErrors = {}

        try {
            const created = await api.createRegistrationKeys({ count: Number(keyCount) })
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

<section class="fade-in">
    <div>
        <h2 class="text-3xl text-slate-900 dark:text-slate-100">Admin</h2>
    </div>

    {#if !auth.user}
        <div
            class="mt-6 rounded-2xl border border-amber-500/40 bg-amber-500/10 p-6 text-sm text-amber-800 dark:text-amber-100"
        >
            Admin functions require login.
        </div>
    {:else if auth.user.role !== 'admin'}
        <div
            class="mt-6 rounded-2xl border border-rose-500/40 bg-rose-500/10 p-6 text-sm text-rose-700 dark:text-rose-200"
        >
            Access denied. Admin account required.
        </div>
    {:else}
        <div class="mt-6">
            <div
                class="inline-flex rounded-full border border-slate-200 bg-white p-1 text-sm dark:border-slate-800/80 dark:bg-slate-900/40"
            >
                <button
                    class={`rounded-full px-4 py-2 transition ${
                        activeTab === 'challenges'
                            ? 'bg-teal-600 text-white'
                            : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
                    }`}
                    onclick={() => (activeTab = 'challenges')}
                >
                    Create Challenge
                </button>
                <button
                    class={`rounded-full px-4 py-2 transition ${
                        activeTab === 'challenge_management'
                            ? 'bg-teal-600 text-white'
                            : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
                    }`}
                    onclick={() => (activeTab = 'challenge_management')}
                >
                    Challenge Management
                </button>
                <button
                    class={`rounded-full px-4 py-2 transition ${
                        activeTab === 'registration_keys'
                            ? 'bg-teal-600 text-white'
                            : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'
                    }`}
                    onclick={() => (activeTab = 'registration_keys')}
                >
                    Registration Keys
                </button>
            </div>

            {#if activeTab === 'challenges'}
                <div
                    class="mt-6 rounded-3xl border border-slate-200 bg-white p-8 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <form
                        class="space-y-5"
                        onsubmit={(event) => {
                            event.preventDefault()
                            submit()
                        }}
                    >
                        <div>
                            <label
                                class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                for="admin-title">Title</label
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
                            <label
                                class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                for="admin-description">Description</label
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
                        <div class="grid gap-4 md:grid-cols-2">
                            <div>
                                <label
                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                    for="admin-category">Category</label
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
                                <label
                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                    for="admin-points">Points</label
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
                                    for="admin-flag">Flag</label
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
                            disabled={loading}
                        >
                            {loading ? 'Creating...' : 'Create Challenge'}
                        </button>
                    </form>
                </div>
            {:else if activeTab === 'challenge_management'}
                <div class="mt-6 space-y-4">
                    <div class="flex items-center justify-between">
                        <button
                            class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
                            onclick={loadChallenges}
                            disabled={challengesLoading}
                        >
                            {challengesLoading ? 'Loading...' : 'Refresh'}
                        </button>
                    </div>

                    {#if challengesErrorMessage}
                        <p
                            class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
                        >
                            {challengesErrorMessage}
                        </p>
                    {/if}
                    {#if challengesSuccessMessage}
                        <p
                            class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-700 dark:text-teal-200"
                        >
                            {challengesSuccessMessage}
                        </p>
                    {/if}

                    {#if challengesLoading}
                        <p class="text-sm text-slate-500 dark:text-slate-400">Loading challenges...</p>
                    {:else}
                        <div
                            class="overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-slate-800/80 dark:bg-slate-900/40"
                        >
                            <div class="overflow-x-auto">
                                <table class="w-full">
                                    <thead
                                        class="border-b border-slate-200 bg-slate-50 dark:border-slate-800 dark:bg-slate-900/60"
                                    >
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
                                                <td
                                                    class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100"
                                                >
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
                                                            수정
                                                        </button>
                                                        <button
                                                            class="text-rose-600 hover:text-rose-700 dark:text-rose-400 dark:hover:text-rose-300"
                                                            onclick={() => deleteChallenge(challenge)}
                                                            disabled={manageLoading}
                                                        >
                                                            삭제
                                                        </button>
                                                    </div>
                                                </td>
                                            </tr>
                                            {#if expandedChallengeId === challenge.id}
                                                <tr class="bg-slate-50/70 dark:bg-slate-900/40">
                                                    <td colspan="6" class="px-6 py-6">
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
                                                                    <p
                                                                        class="mt-2 text-xs text-rose-600 dark:text-rose-300"
                                                                    >
                                                                        title: {manageFieldErrors.title}
                                                                    </p>
                                                                {/if}
                                                            </div>
                                                            <div>
                                                                <label
                                                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                                    for={`manage-description-${challenge.id}`}
                                                                    >Description</label
                                                                >
                                                                <textarea
                                                                    id={`manage-description-${challenge.id}`}
                                                                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                                    rows="5"
                                                                    bind:value={editDescription}
                                                                ></textarea>
                                                                {#if manageFieldErrors.description}
                                                                    <p
                                                                        class="mt-2 text-xs text-rose-600 dark:text-rose-300"
                                                                    >
                                                                        description: {manageFieldErrors.description}
                                                                    </p>
                                                                {/if}
                                                            </div>
                                                            <div class="grid gap-4 md:grid-cols-2">
                                                                <div>
                                                                    <label
                                                                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                                        for={`manage-category-${challenge.id}`}
                                                                        >Category</label
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
                                                                        <p
                                                                            class="mt-2 text-xs text-rose-600 dark:text-rose-300"
                                                                        >
                                                                            category: {manageFieldErrors.category}
                                                                        </p>
                                                                    {/if}
                                                                </div>
                                                                <div>
                                                                    <label
                                                                        class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                                                        for={`manage-points-${challenge.id}`}
                                                                        >Points</label
                                                                    >
                                                                    <input
                                                                        id={`manage-points-${challenge.id}`}
                                                                        class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                                                        type="number"
                                                                        min="1"
                                                                        bind:value={editPoints}
                                                                    />
                                                                    {#if manageFieldErrors.points}
                                                                        <p
                                                                            class="mt-2 text-xs text-rose-600 dark:text-rose-300"
                                                                        >
                                                                            points: {manageFieldErrors.points}
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
                                                    colspan="6"
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
            {:else}
                <div
                    class="mt-6 rounded-3xl border border-slate-200 bg-white p-8 dark:border-slate-800/80 dark:bg-slate-900/40"
                >
                    <form
                        class="space-y-4"
                        onsubmit={(event) => {
                            event.preventDefault()
                            submitKeys()
                        }}
                    >
                        <div class="grid gap-4 md:grid-cols-[1fr_auto]">
                            <div>
                                <label
                                    class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                                    for="admin-key-count">Count</label
                                >
                                <input
                                    id="admin-key-count"
                                    class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                                    type="number"
                                    min="1"
                                    bind:value={keyCount}
                                />
                                {#if createKeysFieldErrors.count}
                                    <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                                        count: {createKeysFieldErrors.count}
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
            {/if}
        </div>
    {/if}
</section>
