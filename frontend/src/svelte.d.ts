declare module '*.svelte' {
    import type { SvelteComponentTyped } from 'svelte'
    export default class Component<Props = {}, Events = {}, Slots = {}> extends SvelteComponentTyped<
        Props,
        Events,
        Slots
    > {}
}
