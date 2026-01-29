<script lang="ts">
    import _ScoreboardTimeline from '../components/ScoreboardTimeline.svelte'
    import _ScoreboardLeaderboard from '../components/ScoreboardLeaderboard.svelte'

    const ScoreboardTimeline = _ScoreboardTimeline
    const ScoreboardLeaderboard = _ScoreboardLeaderboard

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let windowMinutes = $state(60)
    let timelineKey = $state(0)
    let viewMode = $state<'users' | 'groups'>('users')

    const reloadTimeline = () => {
        timelineKey++
    }
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Scoreboard</h2>
        </div>
        <div class="flex flex-wrap gap-3 text-xs text-slate-700 dark:text-slate-300">
            <div
                class="flex items-center gap-2 rounded-full border border-slate-300 bg-white px-3 py-2 dark:border-slate-800/70 dark:bg-slate-900/40"
            >
                <button
                    class={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                        viewMode === 'users'
                            ? 'bg-teal-500/20 text-teal-700 dark:text-teal-200'
                            : 'text-slate-600 hover:text-teal-600 dark:text-slate-400 dark:hover:text-teal-200'
                    }`}
                    onclick={() => {
                        viewMode = 'users'
                        reloadTimeline()
                    }}
                >
                    Users
                </button>
                <button
                    class={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                        viewMode === 'groups'
                            ? 'bg-teal-500/20 text-teal-700 dark:text-teal-200'
                            : 'text-slate-600 hover:text-teal-600 dark:text-slate-400 dark:hover:text-teal-200'
                    }`}
                    onclick={() => {
                        viewMode = 'groups'
                        reloadTimeline()
                    }}
                >
                    Group / Organization
                </button>
            </div>
            <label
                class="flex items-center gap-2 rounded-full border border-slate-300 bg-white px-3 py-2 dark:border-slate-800/70 dark:bg-slate-900/40"
            >
                Window
                <input
                    class="w-20 bg-transparent text-right text-xs text-slate-900 focus:outline-none dark:text-slate-200"
                    type="number"
                    min="10"
                    max="1440"
                    bind:value={windowMinutes}
                    onchange={reloadTimeline}
                />
                <span class="text-slate-600 dark:text-slate-400">min</span>
            </label>
        </div>
    </div>

    <div class="mt-6 grid min-w-0 grid-cols-1 gap-6">
        {#key timelineKey}
            <ScoreboardTimeline {windowMinutes} mode={viewMode} />
        {/key}
        <ScoreboardLeaderboard mode={viewMode} />
    </div>
</section>
