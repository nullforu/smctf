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
        error: 'border-danger/40 bg-danger/10 text-danger ',
        success: 'border-accent/40 bg-accent/10 text-accent-strong ',
        warning: 'border-warning/40 bg-warning/10 text-warning-strong ',
        info: 'border-border/70 bg-surface/60 text-text   ',
    }
</script>

<p class={`rounded-xl border px-4 py-2 text-xs ${styles[variant]} ${className}`}>
    {#if message}
        {message}
    {:else}
        {@render children?.()}
    {/if}
</p>
