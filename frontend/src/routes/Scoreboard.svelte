<script lang="ts">
    import ScoreboardTimeline from '../components/ScoreboardTimeline.svelte'
    import ScoreboardLeaderboard from '../components/ScoreboardLeaderboard.svelte'
    import { t } from '../lib/i18n'

    let viewMode = $state<'users' | 'teams'>('users')
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-text">{$t('scoreboard.title')}</h2>
        </div>
        <div class="flex flex-wrap gap-3 text-xs text-text">
            <div class="flex items-center gap-2 rounded-full border border-border bg-surface px-3 py-2">
                <button
                    class={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                        viewMode === 'users'
                            ? 'bg-accent/20 text-accent-strong '
                            : 'text-text-muted hover:text-accent  '
                    }`}
                    onclick={() => {
                        viewMode = 'users'
                    }}
                >
                    {$t('scoreboard.users')}
                </button>
                <button
                    class={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                        viewMode === 'teams'
                            ? 'bg-accent/20 text-accent-strong '
                            : 'text-text-muted hover:text-accent  '
                    }`}
                    onclick={() => {
                        viewMode = 'teams'
                    }}
                >
                    {$t('scoreboard.teams')}
                </button>
            </div>
        </div>
    </div>

    <div class="mt-6 grid min-w-0 grid-cols-1 gap-6">
        <ScoreboardTimeline mode={viewMode} />
        <ScoreboardLeaderboard mode={viewMode} />
    </div>
</section>
