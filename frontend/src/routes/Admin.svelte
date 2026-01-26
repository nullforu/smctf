<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import { api } from '../lib/api'
    import { formatApiError, type FieldErrors } from '../lib/utils'

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let title = $state('')
    let description = $state('')
    let points = $state(100)
    let flag = $state('')
    let isActive = $state(true)

    let loading = $state(false)
    let errorMessage = $state('')
    let fieldErrors: FieldErrors = $state({})
    let successMessage = $state('')
    let auth = $state(get(authStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
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
                points: Number(points),
                flag,
                is_active: isActive,
            })

            successMessage = `Challenge "${created.title}" (ID ${created.id}) created successfully`
            title = ''
            description = ''
            points = 100
            flag = ''
            isActive = true
        } catch (error) {
            const formatted = formatApiError(error)

            errorMessage = formatted.message
            fieldErrors = formatted.fieldErrors
        } finally {
            loading = false
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
                            <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">points: {fieldErrors.points}</p>
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
                            <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">flag: {fieldErrors.flag}</p>
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
    {/if}
</section>
