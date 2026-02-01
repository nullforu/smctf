<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore } from '../lib/stores'
    import CreateChallenge_ from './admin/CreateChallenge.svelte'
    import ChallengeManagement_ from './admin/ChallengeManagement.svelte'
    import RegistrationKeys_ from './admin/RegistrationKeys.svelte'
    import Teams_ from './admin/Teams.svelte'
    import SiteConfig_ from './admin/SiteConfig.svelte'

    const CreateChallenge = CreateChallenge_
    const ChallengeManagement = ChallengeManagement_
    const RegistrationKeys = RegistrationKeys_
    const Teams = Teams_
    const SiteConfig = SiteConfig_

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let activeTab = $state<'challenges' | 'challenge_management' | 'registration_keys' | 'teams' | 'site_config'>(
        'challenges',
    )
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
        <h2 class="text-2xl font-semibold text-slate-900 dark:text-slate-100 md:text-3xl">Admin</h2>
    </div>

    {#if !auth.user}
        <div
            class="rounded-2xl border border-amber-500/40 bg-amber-500/10 p-4 text-sm text-amber-800 dark:text-amber-100 md:p-6"
        >
            Admin functions require login.
        </div>
    {:else if auth.user.role !== 'admin'}
        <div
            class="rounded-2xl border border-rose-500/40 bg-rose-500/10 p-4 text-sm text-rose-700 dark:text-rose-200 md:p-6"
        >
            Access denied. Admin account required.
        </div>
    {:else}
        <div class="mb-4 flex items-center gap-3">
            <select
                class="flex-1 rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400 md:hidden"
                bind:value={activeTab}
            >
                <option value="challenges">Create Challenge</option>
                <option value="challenge_management">Challenge Management</option>
                <option value="teams">Teams</option>
                <option value="registration_keys">Registration Keys</option>
                <option value="site_config">Site Configuration</option>
            </select>

            {#if !showSidebar}
                <select
                    class="hidden flex-1 rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 focus:border-teal-500 focus:outline-none dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-100 dark:focus:border-teal-400 md:block"
                    bind:value={activeTab}
                >
                    <option value="challenges">Create Challenge</option>
                    <option value="challenge_management">Challenge Management</option>
                    <option value="teams">Teams</option>
                    <option value="registration_keys">Registration Keys</option>
                    <option value="site_config">Site Configuration</option>
                </select>
            {/if}

            <button
                class="hidden text-sm text-slate-700 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white md:block"
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
                    <div
                        class="rounded-2xl border border-slate-200 bg-white p-2 dark:border-slate-800/80 dark:bg-slate-900/40"
                    >
                        <button
                            class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                activeTab === 'challenges'
                                    ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800/60 dark:text-slate-100'
                                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800/40'
                            }`}
                            onclick={() => (activeTab = 'challenges')}
                        >
                            Create Challenge
                        </button>
                        <button
                            class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                activeTab === 'challenge_management'
                                    ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800/60 dark:text-slate-100'
                                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800/40'
                            }`}
                            onclick={() => (activeTab = 'challenge_management')}
                        >
                            Challenge Management
                        </button>
                        <button
                            class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                activeTab === 'registration_keys'
                                    ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800/60 dark:text-slate-100'
                                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800/40'
                            }`}
                            onclick={() => (activeTab = 'registration_keys')}
                        >
                            Registration Keys
                        </button>
                        <button
                            class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                activeTab === 'teams'
                                    ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800/60 dark:text-slate-100'
                                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800/40'
                            }`}
                            onclick={() => (activeTab = 'teams')}
                        >
                            Teams
                        </button>
                        <button
                            class={`flex w-full items-center rounded-lg px-4 py-2.5 text-left text-sm transition ${
                                activeTab === 'site_config'
                                    ? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-800/60 dark:text-slate-100'
                                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800/40'
                            }`}
                            onclick={() => (activeTab = 'site_config')}
                        >
                            Site Configuration
                        </button>
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
