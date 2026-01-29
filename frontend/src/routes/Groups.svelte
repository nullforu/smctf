<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import type { GroupSummary } from '../lib/types'
    import { formatApiError } from '../lib/utils'
    import { navigate as _navigate } from '../lib/router'

    const navigate = _navigate

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let groups: GroupSummary[] = $state([])
    let filteredGroups: GroupSummary[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let searchQuery = $state('')

    const loadGroups = async () => {
        loading = true
        errorMessage = ''

        try {
            groups = await api.groups()
            filteredGroups = groups.sort((a, b) => a.id - b.id)
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    $effect(() => {
        if (searchQuery.trim() === '') {
            filteredGroups = groups
        } else {
            const query = searchQuery.toLowerCase()
            filteredGroups = groups.filter(
                (group) => group.name.toLowerCase().includes(query) || group.id.toString().includes(query),
            )
        }
    })

    onMount(loadGroups)
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Groups</h2>
            <p class="mt-1 text-sm text-slate-600 dark:text-slate-400">Browse groups and their stats.</p>
        </div>
    </div>

    <div class="mt-6">
        <input
            type="text"
            placeholder="Search by group name or ID..."
            bind:value={searchQuery}
            class="w-full rounded-xl border border-slate-300 bg-white px-4 py-2.5 text-sm text-slate-900 placeholder-slate-500 transition focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-500/20 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-100 dark:placeholder-slate-400"
        />
    </div>

    {#if loading}
        <p class="mt-6 text-sm text-slate-600 dark:text-slate-400">Loading...</p>
    {:else if errorMessage}
        <p class="mt-6 text-sm text-rose-700 dark:text-rose-200">{errorMessage}</p>
    {:else}
        <div class="mt-6">
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
                                    Group
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Members
                                </th>
                                <th
                                    class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Total Score
                                </th>
                                <th
                                    class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-600 dark:text-slate-400"
                                >
                                    Action
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-slate-200 dark:divide-slate-800">
                            {#each filteredGroups as group}
                                <tr
                                    class="transition hover:bg-slate-50 dark:hover:bg-slate-900/60 cursor-pointer"
                                    onclick={() => navigate(`/groups/${group.id}`)}
                                >
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                        {group.id}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-900 dark:text-slate-100">
                                        {group.name}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-slate-700 dark:text-slate-300">
                                        {group.member_count}
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-sm text-teal-600 dark:text-teal-200">
                                        {group.total_score} pts
                                    </td>
                                    <td class="whitespace-nowrap px-6 py-4 text-right text-sm">
                                        <button
                                            class="text-teal-600 hover:text-teal-700 dark:text-teal-400 dark:hover:text-teal-300"
                                            onclick={() => navigate(`/groups/${group.id}`)}
                                        >
                                            View
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                            {#if filteredGroups.length === 0}
                                <tr>
                                    <td
                                        colspan="5"
                                        class="px-6 py-8 text-center text-sm text-slate-600 dark:text-slate-400"
                                    >
                                        {searchQuery ? 'No results found.' : 'No groups found.'}
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                </div>
            </div>

            {#if filteredGroups.length > 0}
                <p class="mt-4 text-sm text-slate-600 dark:text-slate-400">
                    {filteredGroups.length} group{filteredGroups.length !== 1 ? 's' : ''}
                    {#if searchQuery}
                        (out of {groups.length})
                    {/if}
                </p>
            {/if}
        </div>
    {/if}
</section>
