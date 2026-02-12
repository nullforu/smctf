<script lang="ts">
    import { get } from 'svelte/store'
    import { authStore, type AuthState } from '../lib/stores'
    import { api } from '../lib/api'
    import type { Stack, UserDetail, SolvedChallenge } from '../lib/types'
    import { formatApiError, formatDateTime, parseRouteId } from '../lib/utils'
    import { navigate } from '../lib/router'
    import ProfileHeader from '../components/user-profile/ProfileHeader.svelte'
    import AccountCard from '../components/user-profile/AccountCard.svelte'
    import ActiveStacksCard from '../components/user-profile/ActiveStacksCard.svelte'
    import SolvedChallengesCard from '../components/user-profile/SolvedChallengesCard.svelte'
    import StatisticsCard from '../components/user-profile/StatisticsCard.svelte'

    interface Props {
        routeParams?: Record<string, string>
    }

    let { routeParams = {} }: Props = $props()

    let user: UserDetail | null = $state(null)
    let solved: SolvedChallenge[] = $state([])
    let loading = $state(false)
    let errorMessage = $state('')
    let auth = $state<AuthState>(get(authStore))
    let stacks = $state<Stack[]>([])
    let stacksLoading = $state(false)
    let stacksError = $state('')
    let stackDeletingId = $state<number | null>(null)

    let editingUsername = $state(false)
    let usernameInput = $state('')
    let savingUsername = $state(false)
    let lastLoadedUserId = $state<number | null>(null)
    let lastStacksLoadedForUserId = $state<number | null>(null)

    const formatDateTimeLocal = formatDateTime
    const formatOptionalDateTime = (value?: string | null) => (value ? formatDateTime(value) : 'N/A')

    $effect(() => {
        const unsubscribe = authStore.subscribe((value) => {
            auth = value
        })
        return unsubscribe
    })

    const routeUserId = $derived(parseRouteId(routeParams.id))
    const isOwnProfile = $derived(auth.user ? !routeUserId || routeUserId === auth.user.id : false)
    const showBackButton = $derived(!!routeParams.id)
    const activeStacks = $derived(
        stacks.filter((stack) => !['stopped', 'failed', 'node_deleted'].includes(stack.status)),
    )
    const targetUserId = $derived(routeUserId ?? auth.user?.id ?? null)
    const totalSolvedPoints = $derived(solved.reduce((sum, item) => sum + item.points, 0))

    const loadUserProfile = async (userId: number) => {
        loading = true
        errorMessage = ''
        user = null
        solved = []

        try {
            const [userDetail, solvedData] = await Promise.all([api.user(userId), api.userSolved(userId)])
            user = userDetail
            solved = solvedData
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            loading = false
        }
    }

    const loadStacks = async () => {
        if (!isOwnProfile) return

        stacksLoading = true
        stacksError = ''

        try {
            stacks = await api.stacks()
        } catch (error) {
            stacksError = formatApiError(error).message
        } finally {
            stacksLoading = false
        }
    }

    const deleteStack = async (challengeId: number) => {
        if (stackDeletingId !== null) return

        stackDeletingId = challengeId
        stacksError = ''

        try {
            await api.deleteStack(challengeId)
            await loadStacks()
        } catch (error) {
            stacksError = formatApiError(error).message
        } finally {
            stackDeletingId = null
        }
    }

    const saveUsername = async () => {
        if (!user) return

        savingUsername = true
        errorMessage = ''

        try {
            const updated = await api.updateMe(usernameInput.trim())
            user = updated
            editingUsername = false
        } catch (error) {
            errorMessage = formatApiError(error).message
        } finally {
            savingUsername = false
        }
    }

    $effect(() => {
        if (user && isOwnProfile) {
            usernameInput = user.username
        }
    })

    $effect(() => {
        if (targetUserId === null) return
        if (lastLoadedUserId === targetUserId) return

        lastLoadedUserId = targetUserId
        loadUserProfile(targetUserId)
    })

    $effect(() => {
        if (!isOwnProfile) {
            lastStacksLoadedForUserId = null
            return
        }

        if (!auth.user) return
        if (lastStacksLoadedForUserId === auth.user.id) return

        lastStacksLoadedForUserId = auth.user.id
        loadStacks()
    })
</script>

<section class="fade-in">
    {#if showBackButton}
        <div class="mb-6">
            <button
                class="inline-flex items-center gap-2 text-sm text-text-muted hover:text-accent"
                onclick={() => navigate('/users')}
            >
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                >
                    <path d="m15 18-6-6 6-6" />
                </svg>
                Back to Users
            </button>
        </div>
    {/if}

    {#if !auth.user}
        <div>
            <h2 class="text-3xl text-text">Profile</h2>
        </div>
        <div class="mt-6 rounded-2xl border border-warning/40 bg-warning/10 p-6 text-sm text-warning-strong">
            Please <a class="underline" href="/login" onclick={(e) => navigate('/login', e)}>login</a> to view your profile.
        </div>
    {:else if loading}
        <div class="rounded-2xl border border-border bg-surface p-8">
            <p class="text-center text-sm text-text-muted">Loading...</p>
        </div>
    {:else if errorMessage}
        <div class="rounded-2xl border border-danger/30 bg-danger/10 p-8">
            <p class="text-center text-sm text-danger">{errorMessage}</p>
        </div>
    {:else if user}
        <div>
            <ProfileHeader {user} />

            {#if isOwnProfile}
                <AccountCard
                    {user}
                    authEmail={auth.user?.email}
                    {savingUsername}
                    onSave={saveUsername}
                    bind:editingUsername
                    bind:usernameInput
                />

                <ActiveStacksCard
                    {activeStacks}
                    {stacksError}
                    {stacksLoading}
                    {stackDeletingId}
                    onRefresh={loadStacks}
                    onDelete={deleteStack}
                    {formatOptionalDateTime}
                />
            {/if}

            <SolvedChallengesCard {solved} formatDateTime={formatDateTimeLocal} />

            {#if solved.length > 0}
                <StatisticsCard totalPoints={totalSolvedPoints} solvedCount={solved.length} />
            {/if}
        </div>
    {/if}
</section>
