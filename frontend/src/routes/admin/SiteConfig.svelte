<script lang="ts">
    import { api } from '../../lib/api'
    import { setConfig } from '../../lib/config'
    import { formatApiError, type FieldErrors } from '../../lib/utils'
    import { onMount } from 'svelte'
    import FormMessage from '../../components/FormMessage.svelte'
    import { t } from '../../lib/i18n'
    import { get } from 'svelte/store'

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
            configSuccessMessage = get(t)('admin.site.saved')
        } catch (error) {
            const formatted = formatApiError(error)
            configErrorMessage = formatted.message
            configFieldErrors = formatted.fieldErrors
        } finally {
            configLoading = false
        }
    }
</script>

<div class="rounded-3xl border border-border bg-surface p-4 md:p-8">
    <div class="flex items-center justify-between">
        <div>
            <h3 class="text-lg text-text">{$t('admin.site.title')}</h3>
            <p class="text-xs text-text-subtle">{$t('admin.site.subtitle')}</p>
        </div>
        <button
            class="text-xs uppercase tracking-wide text-text-subtle hover:text-text"
            onclick={loadSiteConfig}
            disabled={configLoading}
        >
            {configLoading ? $t('common.loading') : $t('common.reload')}
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
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-header-title"
                >{$t('admin.site.headerTitle')}</label
            >
            <input
                id="admin-header-title"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                type="text"
                bind:value={headerTitle}
                placeholder={$t('admin.site.headerTitlePlaceholder')}
            />
            {#if configFieldErrors.header_title}
                <p class="mt-2 text-xs text-danger">
                    {$t('admin.site.headerTitle')}: {configFieldErrors.header_title}
                </p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-header-description"
                >{$t('admin.site.headerDescription')}</label
            >
            <textarea
                id="admin-header-description"
                class="mt-2 h-28 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                bind:value={headerDescription}
                placeholder={$t('admin.site.headerDescriptionPlaceholder')}
            ></textarea>
            {#if configFieldErrors.header_description}
                <p class="mt-2 text-xs text-danger">
                    {$t('admin.site.headerDescription')}: {configFieldErrors.header_description}
                </p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-site-title"
                >{$t('admin.site.siteTitle')}</label
            >
            <input
                id="admin-site-title"
                class="mt-2 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                type="text"
                bind:value={configTitle}
                placeholder={$t('admin.site.siteTitlePlaceholder')}
            />
            {#if configFieldErrors.title}
                <p class="mt-2 text-xs text-danger">{$t('admin.site.siteTitle')}: {configFieldErrors.title}</p>
            {/if}
        </div>
        <div>
            <label class="text-xs uppercase tracking-wide text-text-muted" for="admin-site-description"
                >{$t('admin.site.description')}</label
            >
            <textarea
                id="admin-site-description"
                class="mt-2 h-32 w-full rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none"
                bind:value={configDescription}
                placeholder={$t('admin.site.siteDescriptionPlaceholder')}
            ></textarea>
            {#if configFieldErrors.description}
                <p class="mt-2 text-xs text-danger">
                    {$t('admin.site.description')}: {configFieldErrors.description}
                </p>
            {/if}
        </div>

        {#if configErrorMessage}
            <FormMessage variant="error" message={configErrorMessage} />
        {/if}
        {#if configSuccessMessage}
            <FormMessage variant="success" message={configSuccessMessage} />
        {/if}

        <button
            class="w-full rounded-xl bg-accent py-3 text-sm text-contrast-foreground transition hover:bg-accent-strong disabled:opacity-60"
            type="submit"
            disabled={configLoading}
        >
            {configLoading ? $t('admin.site.saving') : $t('admin.site.saveButton')}
        </button>
    </form>
</div>
