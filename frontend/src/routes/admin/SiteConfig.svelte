<script lang="ts">
    import { api } from '../../lib/api'
    import { setConfig } from '../../lib/config'
    import { formatApiError, type FieldErrors } from '../../lib/utils'
    import { onMount } from 'svelte'

    let configTitle = $state('')
    let configDescription = $state('')
    let headerTitle = $state('')
    let headerDescription = $state('')
    let configLoading = $state(false)
    let configErrorMessage = $state('')
    let configSuccessMessage = $state('')
    let configFieldErrors: FieldErrors = $state({})

    onMount(() => {
        loadSiteConfig()
    })

    const loadSiteConfig = async () => {
        configLoading = true
        configErrorMessage = ''
        configSuccessMessage = ''
        configFieldErrors = {}

        try {
            const response = await api.config()
            configTitle = response.title
            configDescription = response.description
            headerTitle = response.header_title
            headerDescription = response.header_description
        } catch (error) {
            const formatted = formatApiError(error)
            configErrorMessage = formatted.message
        } finally {
            configLoading = false
        }
    }

    const saveSiteConfig = async () => {
        configLoading = true
        configErrorMessage = ''
        configSuccessMessage = ''
        configFieldErrors = {}

        try {
            const response = await api.updateAdminConfig({
                title: configTitle,
                description: configDescription,
                header_title: headerTitle,
                header_description: headerDescription,
            })
            configTitle = response.title
            configDescription = response.description
            headerTitle = response.header_title
            headerDescription = response.header_description
            setConfig(response)
            configSuccessMessage = 'Configuration saved.'
        } catch (error) {
            const formatted = formatApiError(error)
            configErrorMessage = formatted.message
            configFieldErrors = formatted.fieldErrors
        } finally {
            configLoading = false
        }
    }
</script>

<div class="rounded-3xl border border-slate-200 bg-white p-4 dark:border-slate-800/80 dark:bg-slate-900/40 md:p-8">
    <div class="flex items-center justify-between">
        <div>
            <h3 class="text-lg text-slate-900 dark:text-slate-100">Site Configuration</h3>
            <p class="text-xs text-slate-500 dark:text-slate-400">Customize the appearance and details of CTF.</p>
        </div>
        <button
            class="text-xs uppercase tracking-wide text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
            onclick={loadSiteConfig}
            disabled={configLoading}
        >
            {configLoading ? 'Loading...' : 'Reload'}
        </button>
    </div>

    <form
        class="mt-6 space-y-4"
        onsubmit={(event) => {
            event.preventDefault()
            saveSiteConfig()
        }}
    >
        <div>
            <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-header-title"
                >Header Title</label
            >
            <input
                id="admin-header-title"
                class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                type="text"
                bind:value={headerTitle}
                placeholder="CTF"
            />
            {#if configFieldErrors.header_title}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                    header_title: {configFieldErrors.header_title}
                </p>
            {/if}
        </div>
        <div>
            <label
                class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                for="admin-header-description">Header Description</label
            >
            <textarea
                id="admin-header-description"
                class="mt-2 h-28 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                bind:value={headerDescription}
                placeholder="Capture The Flag"
            ></textarea>
            {#if configFieldErrors.header_description}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                    header_description: {configFieldErrors.header_description}
                </p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400" for="admin-site-title"
                >Title</label
            >
            <input
                id="admin-site-title"
                class="mt-2 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                type="text"
                bind:value={configTitle}
                placeholder="Welcome to SMCTF."
            />
            {#if configFieldErrors.title}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">title: {configFieldErrors.title}</p>
            {/if}
        </div>
        <div>
            <label
                class="text-xs uppercase tracking-wide text-slate-600 dark:text-slate-400"
                for="admin-site-description">Description</label
            >
            <textarea
                id="admin-site-description"
                class="mt-2 h-32 w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400"
                bind:value={configDescription}
                placeholder="Check out the repository for setup instructions."
            ></textarea>
            {#if configFieldErrors.description}
                <p class="mt-2 text-xs text-rose-600 dark:text-rose-300">
                    description: {configFieldErrors.description}
                </p>
            {/if}
        </div>

        {#if configErrorMessage}
            <p
                class="rounded-xl border border-rose-500/40 bg-rose-500/10 px-4 py-2 text-xs text-rose-700 dark:text-rose-200"
            >
                {configErrorMessage}
            </p>
        {/if}
        {#if configSuccessMessage}
            <p
                class="rounded-xl border border-teal-500/40 bg-teal-500/10 px-4 py-2 text-xs text-teal-700 dark:text-teal-200"
            >
                {configSuccessMessage}
            </p>
        {/if}

        <button
            class="w-full rounded-xl bg-teal-600 py-3 text-sm text-white transition hover:bg-teal-700 disabled:opacity-60 dark:bg-teal-500/30 dark:text-teal-100 dark:hover:bg-teal-500/40"
            type="submit"
            disabled={configLoading}
        >
            {configLoading ? 'Saving...' : 'Save Configuration'}
        </button>
    </form>
</div>
