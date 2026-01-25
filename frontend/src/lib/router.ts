export const navigate = (path: string) => {
    if (typeof window === 'undefined') return

    const normalized = path.startsWith('/') ? path : `/${path}`
    if (window.location.pathname === normalized) return

    window.history.pushState({}, '', normalized)
    window.dispatchEvent(new PopStateEvent('popstate'))
}
