<script lang="ts">
    import { marked } from 'marked'
    import DOMPurify from 'dompurify'

    interface Props {
        content: string
        class?: string
    }

    let { content, class: className = '' }: Props = $props()

    let html = $derived.by(() => {
        if (!content) return ''
        const rawHtml = marked.parse(content, { async: false }) as string
        return DOMPurify.sanitize(rawHtml)
    })
</script>

<div class="markdown {className}">
    {@html html}
</div>

<style>
    .markdown :global(h1) {
        @apply mb-3 text-xl font-bold sm:mb-4 sm:text-2xl md:text-3xl;
    }

    .markdown :global(h2) {
        @apply mb-2.5 text-lg font-semibold sm:mb-3 sm:text-xl md:text-2xl;
    }

    .markdown :global(h3) {
        @apply mb-2 text-base font-semibold sm:text-lg md:text-xl;
    }

    .markdown :global(p) {
        @apply mb-3 leading-relaxed sm:mb-4;
    }

    .markdown :global(a) {
        @apply text-accent hover:underline;
    }

    .markdown :global(ul) {
        @apply mb-3 ml-4 list-disc sm:mb-4 sm:ml-6;
    }

    .markdown :global(ol) {
        @apply mb-3 ml-4 list-decimal sm:mb-4 sm:ml-6;
    }

    .markdown :global(li) {
        @apply mb-1 leading-relaxed;
    }

    .markdown :global(code) {
        @apply rounded bg-surface-subtle px-1.5 py-0.5 text-sm  sm:text-base;
    }

    .markdown :global(pre) {
        @apply mb-3 overflow-x-auto rounded-lg bg-surface-subtle p-3 text-sm  sm:mb-4 sm:p-4 sm:text-base;
    }

    .markdown :global(pre code) {
        @apply bg-transparent p-0;
    }

    .markdown :global(blockquote) {
        @apply mb-4 border-l-4 border-border pl-4 italic;
    }

    .markdown :global(strong) {
        @apply font-semibold;
    }

    .markdown :global(em) {
        @apply italic;
    }
</style>
