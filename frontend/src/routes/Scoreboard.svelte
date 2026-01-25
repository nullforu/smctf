<script lang="ts">
    import ScoreboardTimeline from '../components/ScoreboardTimeline.svelte'
    import ScoreboardLeaderboard from '../components/ScoreboardLeaderboard.svelte'

    let limit = $state(50)
    let interval = $state(10)
    let windowMinutes = $state(60)
    let timelineKey = $state(0)

    const TimelineComponent = ScoreboardTimeline
    const LeaderboardComponent = ScoreboardLeaderboard

    const reloadTimeline = () => {
        timelineKey++
    }
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-100">Scoreboard</h2>
        </div>
        <div class="flex flex-wrap gap-3 text-xs text-slate-300">
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Limit
                <input
                    class="w-16 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="1"
                    max="200"
                    bind:value={limit}
                    onchange={reloadTimeline}
                />
            </label>
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Interval
                <input
                    class="w-16 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="1"
                    max="120"
                    bind:value={interval}
                    onchange={reloadTimeline}
                />
                <span class="text-slate-400">min</span>
            </label>
            <label class="flex items-center gap-2 rounded-full border border-slate-800/70 bg-slate-900/40 px-3 py-2">
                Window
                <input
                    class="w-20 bg-transparent text-right text-xs text-slate-200 focus:outline-none"
                    type="number"
                    min="10"
                    max="1440"
                    bind:value={windowMinutes}
                    onchange={reloadTimeline}
                />
                <span class="text-slate-400">min</span>
            </label>
        </div>
    </div>

    <div class="mt-6 grid min-w-0 grid-cols-1 gap-6">
        {#key timelineKey}
            <TimelineComponent {interval} {limit} {windowMinutes} />
        {/key}
        <LeaderboardComponent {limit} />
    </div>
</section>
