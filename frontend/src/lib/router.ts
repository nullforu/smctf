export const navigate = (path: string, event?: Event) => {
    if (typeof window === 'undefined') return

    event?.preventDefault()

    const normalized = path.startsWith('/') ? path : `/${path}`
    if (window.location.pathname === normalized) return

    window.history.pushState({}, '', normalized)
    window.dispatchEvent(new PopStateEvent('popstate'))
}
