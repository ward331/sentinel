import { useState, useEffect, useCallback } from 'react'
import { fetchHealth, fetchMetrics, fetchProviders, fetchProviderHealth } from '../../api/client'
import type { HealthResponse, MetricsResponse, ProviderInfo } from '../../types/sentinel'
import { Activity, Server, Clock, Radio, Loader2, RefreshCw, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react'

const STATUS_COLOR = {
  healthy: 'text-emerald-400',
  degraded: 'text-yellow-400',
  unhealthy: 'text-red-400',
} as const

const STATUS_BG = {
  healthy: 'bg-emerald-400',
  degraded: 'bg-yellow-400',
  unhealthy: 'bg-red-400',
} as const

const STATUS_BADGE_BG = {
  healthy: 'bg-emerald-900/50 text-emerald-400 border-emerald-800',
  degraded: 'bg-yellow-900/50 text-yellow-400 border-yellow-800',
  unhealthy: 'bg-red-900/50 text-red-400 border-red-800',
} as const

const TIER_BADGE: Record<string, string> = {
  free: 'bg-gray-700 text-gray-300',
  paid: 'bg-amber-900/50 text-amber-400',
  premium: 'bg-purple-900/50 text-purple-400',
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'healthy': return <CheckCircle2 className="w-4 h-4 text-emerald-400" />
    case 'degraded': return <AlertTriangle className="w-4 h-4 text-yellow-400" />
    case 'unhealthy': return <XCircle className="w-4 h-4 text-red-400" />
    default: return <Activity className="w-4 h-4 text-gray-500" />
  }
}

function formatUptime(s: number): string {
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function formatInterval(s: number): string {
  if (s >= 3600) return `${Math.floor(s / 3600)}h`
  if (s >= 60) return `${Math.floor(s / 60)}m`
  return `${s}s`
}

export function ProviderHealth() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null)
  const [providers, setProviders] = useState<ProviderInfo[]>([])
  const [_providerHealth, setProviderHealth] = useState<Record<string, unknown> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)

  const load = useCallback(async () => {
    try {
      const [h, m, p, ph] = await Promise.allSettled([
        fetchHealth(),
        fetchMetrics(),
        fetchProviders(),
        fetchProviderHealth(),
      ])
      if (h.status === 'fulfilled') setHealth(h.value)
      if (m.status === 'fulfilled') setMetrics(m.value)
      if (p.status === 'fulfilled') setProviders(p.value.providers || [])
      if (ph.status === 'fulfilled') setProviderHealth(ph.value)
      setError(null)
      setLastRefresh(new Date())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const timer = setInterval(load, 30000)
    return () => clearInterval(timer)
  }, [load])

  const enabledCount = providers.filter(p => p.enabled).length
  const disabledCount = providers.filter(p => !p.enabled).length

  const providersByTier: Record<string, ProviderInfo[]> = {}
  for (const p of providers) {
    const tier = p.tier || 'free'
    if (!providersByTier[tier]) providersByTier[tier] = []
    providersByTier[tier].push(p)
  }
  const tierOrder = ['free', 'paid', 'premium']
  const sortedTiers = Object.keys(providersByTier).sort((a, b) => {
    const ai = tierOrder.indexOf(a)
    const bi = tierOrder.indexOf(b)
    return (ai === -1 ? 99 : ai) - (bi === -1 ? 99 : bi)
  })

  // Find max events for bar chart
  const eventsEntries = metrics?.events_by_provider
    ? Object.entries(metrics.events_by_provider).sort(([, a], [, b]) => b - a)
    : []
  const maxEvents = eventsEntries.length > 0 ? eventsEntries[0][1] : 0

  return (
    <div className="p-4 space-y-5 overflow-y-auto h-full">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">System Health</h2>
        <div className="flex items-center gap-2">
          {lastRefresh && (
            <span className="text-[10px] text-gray-600">
              {lastRefresh.toLocaleTimeString()}
            </span>
          )}
          <button
            onClick={load}
            className="p-1.5 rounded hover:bg-gray-800 text-gray-500 hover:text-gray-300 transition-colors"
            title="Refresh"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300">{error}</div>
      )}

      {loading && !health && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 text-gray-500 animate-spin" />
        </div>
      )}

      {health && (
        <>
          {/* ── System Overview Card ── */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <div className="flex items-center gap-3 mb-4">
              <Server className="w-5 h-5 text-gray-400" />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className={`text-sm font-bold ${STATUS_COLOR[health.status]}`}>
                    {health.status.toUpperCase()}
                  </span>
                  <span className={`text-[10px] px-2 py-0.5 rounded-full border ${STATUS_BADGE_BG[health.status]}`}>
                    {health.status}
                  </span>
                </div>
                {health.version && (
                  <p className="text-[10px] text-gray-600 mt-0.5">v{health.version}</p>
                )}
              </div>
            </div>

            <div className="grid grid-cols-3 gap-3">
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <Clock className="w-4 h-4 text-gray-500 mx-auto mb-1" />
                <p className="text-sm font-medium text-gray-200">{formatUptime(health.uptime_seconds)}</p>
                <p className="text-[10px] text-gray-500">Uptime</p>
              </div>
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <Activity className="w-4 h-4 text-gray-500 mx-auto mb-1" />
                <p className="text-sm font-medium text-gray-200">
                  {metrics ? metrics.events_processed_total.toLocaleString() : '-'}
                </p>
                <p className="text-[10px] text-gray-500">Total Events</p>
              </div>
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <Radio className="w-4 h-4 text-gray-500 mx-auto mb-1" />
                <p className="text-sm font-medium text-gray-200">{enabledCount}<span className="text-gray-600">/{providers.length}</span></p>
                <p className="text-[10px] text-gray-500">Providers</p>
              </div>
            </div>
          </div>

          {/* ── Provider Count Summary ── */}
          <div className="flex items-center gap-3 text-xs">
            <span className="text-gray-500">{providers.length} providers</span>
            <span className="text-gray-700">|</span>
            <span className="text-emerald-400">{enabledCount} active</span>
            {disabledCount > 0 && (
              <>
                <span className="text-gray-700">|</span>
                <span className="text-gray-500">{disabledCount} disabled</span>
              </>
            )}
          </div>

          {/* ── Provider Grid (by tier) ── */}
          {sortedTiers.map(tier => (
            <div key={tier}>
              <h3 className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider mb-2 flex items-center gap-2">
                <span className={`px-1.5 py-0.5 rounded text-[10px] ${TIER_BADGE[tier] || 'bg-gray-700 text-gray-400'}`}>
                  {tier}
                </span>
                <span className="text-gray-600">{providersByTier[tier].length} providers</span>
              </h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {providersByTier[tier].map(provider => {
                  const statusKey = (provider.status || 'healthy') as keyof typeof STATUS_BG
                  return (
                    <div key={provider.name} className="bg-gray-800 rounded-lg p-3 border border-gray-800 hover:border-gray-700 transition-colors">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2 min-w-0">
                          <span className={`w-2 h-2 rounded-full flex-shrink-0 ${STATUS_BG[statusKey] || 'bg-gray-500'}`} />
                          <span className="text-xs font-medium text-gray-200 truncate">
                            {provider.display_name || provider.name}
                          </span>
                        </div>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded ${provider.enabled ? 'bg-emerald-900/40 text-emerald-400' : 'bg-gray-700 text-gray-500'}`}>
                          {provider.enabled ? 'ON' : 'OFF'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between text-[10px]">
                        <span className="text-gray-500">
                          {provider.events_last_hour != null ? `${provider.events_last_hour} events/hr` : 'no data'}
                        </span>
                        <span className="text-gray-600">
                          {provider.interval_seconds ? `every ${formatInterval(provider.interval_seconds)}` : ''}
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          ))}

          {/* ── Health Checks Panel ── */}
          {health.checks && Object.keys(health.checks).length > 0 && (
            <div>
              <h3 className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider mb-2">Health Checks</h3>
              <div className="space-y-1.5">
                {Object.entries(health.checks).map(([name, check]) => (
                  <div key={name} className="bg-gray-800 rounded-lg px-3 py-2.5 flex items-center gap-3">
                    <StatusIcon status={check.status} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-gray-300 capitalize">{name}</span>
                        <span className={`text-[10px] font-medium ${STATUS_COLOR[check.status] || 'text-gray-400'}`}>
                          {check.status}
                        </span>
                      </div>
                      {check.message && (
                        <p className="text-[10px] text-gray-500 mt-0.5 truncate">{check.message}</p>
                      )}
                    </div>
                    {check.duration_ms != null && (
                      <span className="text-[10px] text-gray-600 tabular-nums flex-shrink-0">
                        {check.duration_ms.toFixed(0)}ms
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ── Events by Provider (bar chart) ── */}
          {eventsEntries.length > 0 && (
            <div>
              <h3 className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider mb-2">Events by Provider</h3>
              <div className="space-y-1.5">
                {eventsEntries.map(([provider, count]) => {
                  const pct = maxEvents > 0 ? (count / maxEvents) * 100 : 0
                  return (
                    <div key={provider} className="bg-gray-800 rounded-lg px-3 py-2">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs text-gray-400 truncate">{provider}</span>
                        <span className="text-xs font-mono text-gray-300 tabular-nums flex-shrink-0 ml-2">
                          {count.toLocaleString()}
                        </span>
                      </div>
                      <div className="h-1.5 bg-gray-900 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-emerald-500/60 rounded-full transition-all duration-500"
                          style={{ width: `${Math.max(pct, 1)}%` }}
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
