<script lang="ts">
    import { onMount } from 'svelte'
    import { api } from '../lib/api'
    import { formatApiError } from '../lib/utils'
    import type { Challenge } from '../lib/types'
    import ChallengeCard from '../components/ChallengeCard.svelte'
    import ChallengeModal from '../components/ChallengeModal.svelte'

    const ChallengeModalComponent = ChallengeModal

    let challenges: Challenge[] = $state([])
    let loading = $state(true)
    let errorMessage = $state('')
    let solvedIds = $state(new Set<number>())
    let selectedChallenge: Challenge | null = $state(null)

    const ChallengeCardComponent = ChallengeCard

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
            const solved = await api.solved()
            solvedIds = new Set(solved.map((item) => item.challenge_id))
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

    onMount(async () => {
        await loadChallenges()
        await loadSolved()
    })
</script>

<section class="fade-in">
    <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
            <h2 class="text-3xl text-slate-900 dark:text-slate-100">Challenges</h2>
        </div>
        <div
            class="rounded-full border border-slate-300 bg-white px-4 py-2 text-xs text-slate-700 dark:border-slate-800/70 dark:bg-slate-900/40 dark:text-slate-300"
        >
            푼 문제 {solvedIds.size} / 전체 {challenges.filter((c) => c.is_active).length}
            {challenges.filter((c) => !c.is_active).length > 0
                ? `(비활성 문제 ${challenges.filter((c) => !c.is_active).length}개)`
                : ''}
        </div>
    </div>

    {#if loading}
        <div
            class="mt-6 rounded-2xl border border-slate-200 bg-white p-8 text-center text-slate-600 dark:border-slate-800/70 dark:bg-slate-900/40 dark:text-slate-400"
        >
            문제를 불러오는 중...
        </div>
    {:else if errorMessage}
        <div
            class="mt-6 rounded-2xl border border-rose-500/40 bg-rose-500/10 p-6 text-sm text-rose-700 dark:text-rose-200"
        >
            {errorMessage}
        </div>
    {:else}
        <div class="mt-6 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {#each challenges as challenge}
                <ChallengeCardComponent
                    {challenge}
                    isSolved={solvedIds.has(challenge.id)}
                    onClick={() => openChallenge(challenge)}
                />
            {/each}
        </div>
    {/if}
</section>

{#if selectedChallenge}
    <ChallengeModalComponent
        challenge={selectedChallenge}
        isSolved={solvedIds.has(selectedChallenge.id)}
        onClose={closeModal}
        onSolved={loadSolved}
    />
{/if}
