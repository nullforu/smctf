<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import CreateChallenge from './admin/CreateChallenge.svelte'
    import ChallengeManagement from './admin/ChallengeManagement.svelte'
    import RegistrationKeys from './admin/RegistrationKeys.svelte'
    import Teams from './admin/Teams.svelte'
    import SiteConfig from './admin/SiteConfig.svelte'

    const adminTabs = [
        { id: 'challenges', label: 'Create Challenge' },
        { id: 'challenge_management', label: 'Challenge Management' },
        { id: 'teams', label: 'Teams' },
        { id: 'registration_keys', label: 'Registration Keys' },
        { id: 'site_config', label: 'Site Configuration' },
    ] as const
    type AdminTabId = (typeof adminTabs)[number]['id']

    let activeTab = $state<AdminTabId>('challenges')
    let showSidebar = $state(false)

    let auth = $state(get(authStore))

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })
</script>

<section class="fade-in">
    <div class="mb-4 md:mb-6">
        <h2 class="text-2xl font-semibold text-text md:text-3xl">Admin</h2>
    </div>

    {#if !auth.user}
        <div class="rounded-2xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning-strong md:p-6">
            Admin functions require login.
        </div>
    {:else if auth.user.role !== 'admin'}
        <div class="rounded-2xl border border-danger/40 bg-danger/10 p-4 text-sm text-danger md:p-6">
            Access denied. Admin account required.
        </div>
    {:else}
        <div class="mb-4 flex items-center gap-3">
            <select
                class="flex-1 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none md:hidden"
                bind:value={activeTab}
            >
                {#each adminTabs as tab}
                    <option value={tab.id}>{tab.label}</option>
                {/each}
            </select>

            {#if !showSidebar}
                <select
                    class="hidden flex-1 rounded-xl border border-border bg-surface px-4 py-3 text-sm text-text focus:border-accent focus:outline-none md:block"
                    bind:value={activeTab}
                >
                    {#each adminTabs as tab}
                        <option value={tab.id}>{tab.label}</option>
                    {/each}
                </select>
            {/if}

            <button
                class="hidden text-sm text-text hover:text-text md:block"
                onclick={() => (showSidebar = !showSidebar)}
                title={showSidebar ? 'Hide sidebar' : 'Show sidebar'}
            >
                {#if showSidebar}
                    <span class="flex items-center gap-2">
                        <span>◀</span>
                        <span>Hide Menu</span>
                    </span>
                {:else}
                    <span class="flex items-center gap-2">
                        <span>▶</span>
                        <span>Show Menu</span>
                    </span>
                {/if}
            </button>
        </div>

        <div class="flex flex-col gap-6 md:flex-row md:gap-8">
            {#if showSidebar}
                <nav class="hidden md:block md:w-64 md:flex-shrink-0">
                    <div class="rounded-2xl border border-border bg-surface p-2">
                        {#each adminTabs as tab}
                            <button
                                class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                    activeTab === tab.id
                                        ? 'bg-surface-subtle font-medium text-text  '
                                        : 'text-text hover:bg-surface-muted  '
                                }`}
                                onclick={() => (activeTab = tab.id)}
                                type="button"
                            >
                                {tab.label}
                            </button>
                        {/each}
                    </div>
                </nav>
            {/if}

            <div class="flex-1 md:min-w-0">
                {#if activeTab === 'challenges'}
                    <CreateChallenge />
                {:else if activeTab === 'challenge_management'}
                    <ChallengeManagement />
                {:else if activeTab === 'registration_keys'}
                    <RegistrationKeys />
                {:else if activeTab === 'teams'}
                    <Teams />
                {:else if activeTab === 'site_config'}
                    <SiteConfig />
                {/if}
            </div>
        </div>
    {/if}
</section>
