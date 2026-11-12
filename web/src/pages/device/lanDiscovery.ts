export type DiscoveredLAN = { networkId: string; displayName: string; device: string; state: string; readOnlyReason?: string; ipv4Prefixes?: string[]; ipv6Prefixes?: string[] }
export class LANDiscoveryError extends Error {
    constructor(message: string, readonly code: 'permission' | 'timeout' | 'unavailable' | 'invalid') { super(message) }
}

export async function discoverLANs(transport: typeof fetch = fetch) {
    const controller = new AbortController(), timer = setTimeout(() => controller.abort(), 8000)
    try {
        const response = await transport('/cgi-bin/luci/istore/lanctrl/v2/networks/', { credentials: 'same-origin', signal: controller.signal })
        // Only an explicitly missing endpoint permits the old default-LAN contract.
        if (response.status === 404 || response.status === 501) return { networks: [{ networkId: 'lan', displayName: 'lan', device: '', state: 'legacy' }], legacy: true }
        if (response.status === 401) throw new LANDiscoveryError('登录已过期，请重新登录', 'permission')
        if (response.status === 403) throw new LANDiscoveryError('没有读取局域网列表的权限', 'permission')
        if (!response.ok) throw new LANDiscoveryError('局域网服务暂不可用，请重试', 'unavailable')
        let body: any
        try { body = await response.json() } catch { throw new LANDiscoveryError('局域网列表格式不正确，请重试', 'invalid') }
        if (!body || typeof body !== 'object') throw new LANDiscoveryError('局域网列表格式不正确，请重试', 'invalid')
        if (body.success === -1001) throw new LANDiscoveryError('登录已过期，请重新登录', 'permission')
        if (body.error || body.success !== undefined && body.success !== 0 || !Array.isArray(body.result?.networks)) throw new LANDiscoveryError('局域网列表不可用，请重试', 'invalid')
        const networks: DiscoveredLAN[] = body.result.networks
        if (networks.some(item => !item || !/^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,31}$/.test(item.networkId) || typeof item.displayName !== 'string') || new Set(networks.map(item => item.networkId)).size !== networks.length) throw new LANDiscoveryError('局域网列表格式不正确，请重试', 'invalid')
        return { networks, legacy: false }
    } catch (reason) {
        const name = (reason as { name?: string })?.name
        if (controller.signal.aborted || name === 'TimeoutError' || name === 'AbortError') throw new LANDiscoveryError('局域网列表读取超时，请重试', 'timeout')
        if (reason instanceof LANDiscoveryError) throw reason
        throw new LANDiscoveryError('无法连接路由器，请检查网络后重试', 'unavailable')
    } finally { clearTimeout(timer) }
}
