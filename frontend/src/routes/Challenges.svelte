<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import type { Challenge } from '../lib/types'
    import ChallengeCard from '../components/ChallengeCard.svelte'
    import ChallengeModal from '../components/ChallengeModal.svelte'

    let challenges: Challenge[] = $state([])
    let loading = $state(true)
    let errorMessage = $state('')
    let solvedIds = $state(new Set<number>())
    let selectedChallenge: Challenge | null = $state(null)

    const activeChallenges = $derived(challenges.filter((challenge) => challenge.is_active))
    const inactiveChallenges = $derived(challenges.filter((challenge) => !challenge.is_active))
    const solvedCount = $derived(solvedIds.size)

    const loadChallenges = async () => {
        loading = true
        errorMessage = ''

        try {
            challenges = await api.challenges()
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const loadSolved = async () => {
        try {
            const me = await api.me()
            if (!me?.id) {
                solvedIds = new Set()
                return
            }

            const teamSolved = await api.teamSolved(me.team_id)
            solvedIds = new Set(teamSolved.map((item) => item.challenge_id))
        } catch {
            solvedIds = new Set()
        }
    }

    const openChallenge = (challenge: Challenge) => {
        selectedChallenge = challenge
    }

    const closeModal = () => {
        selectedChallenge = null
    }

    onMount(() => {
        void Promise.all([loadChallenges(), loadSolved()])
    })
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-text">Challenges</h2>
        </div>
        <div class="rounded-full border border-border bg-surface px-4 py-2 text-xs text-text">
            Solved {solvedCount} / {activeChallenges.length}
            {inactiveChallenges.length > 0 ? `(${inactiveChallenges.length} inactive)` : ''}
        </div>
    </div>

    {#if loading}
        <div class="mt-6 rounded-2xl border border-border bg-surface p-8 text-center text-text-muted">
            Loading challenges...
        </div>
    {:else if errorMessage}
        <div class="mt-6 rounded-2xl border border-danger/40 bg-danger/10 p-6 text-sm text-danger">
            {errorMessage}
        </div>
    {:else}
        <div class="mt-6 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {#each challenges as challenge}
                <ChallengeCard
                    {challenge}
                    isSolved={solvedIds.has(challenge.id)}
                    onClick={() => openChallenge(challenge)}
                />
            {/each}
        </div>
    {/if}
</section>

{#if selectedChallenge}
    <ChallengeModal
        challenge={selectedChallenge}
        isSolved={solvedIds.has(selectedChallenge.id)}
        onClose={closeModal}
        onSolved={loadSolved}
    />
{/if}
