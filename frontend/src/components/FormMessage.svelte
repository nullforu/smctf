<script lang="ts">
    type Variant = 'error' | 'success' | 'warning' | 'info'

    interface Props {
        variant?: Variant
        message?: string
        className?: string
    }

    import type { Snippet } from 'svelte'

    let { variant = 'error', message, className = '', children }: Props & { children?: Snippet } = $props()

    const styles: Record<Variant, string> = {
        error: 'border-rose-500/40 bg-rose-500/10 text-rose-700 dark:text-rose-200',
        success: 'border-teal-500/40 bg-teal-500/10 text-teal-700 dark:text-teal-200',
        warning: 'border-amber-500/40 bg-amber-500/10 text-amber-800 dark:text-amber-100',
        info: 'border-slate-300/70 bg-slate-100/60 text-slate-700 dark:border-slate-700/70 dark:bg-slate-900/40 dark:text-slate-300',
    }
</script>

<p class={`rounded-xl border px-4 py-2 text-xs ${styles[variant]} ${className}`}>
    {#if message}
        {message}
    {:else}
        {@render children?.()}
    {/if}
</p>
