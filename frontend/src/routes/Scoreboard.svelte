<script lang="ts">
    import ScoreboardTimeline from '../components/ScoreboardTimeline.svelte'
    import ScoreboardLeaderboard from '../components/ScoreboardLeaderboard.svelte'

    let viewMode = $state<'users' | 'teams'>('users')
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
                    }}
                >
                    Users
                </button>
                <button
                    class={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                        viewMode === 'teams'
                            ? 'bg-teal-500/20 text-teal-700 dark:text-teal-200'
                            : 'text-slate-600 hover:text-teal-600 dark:text-slate-400 dark:hover:text-teal-200'
                    }`}
                    onclick={() => {
                        viewMode = 'teams'
                    }}
                >
                    Team
                </button>
            </div>
        </div>
    </div>

    <div class="mt-6 grid min-w-0 grid-cols-1 gap-6">
        <ScoreboardTimeline mode={viewMode} />
        <ScoreboardLeaderboard mode={viewMode} />
    </div>
</section>
