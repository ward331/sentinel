import { useState, useEffect } from 'react'
import { fetchHealth, fetchMetrics } from '../../api/client'
import type { HealthResponse, MetricsResponse } from '../../types/sentinel'
import { Activity, Server, Clock, AlertTriangle } from 'lucide-react'

export function ProviderHealth() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let timer: ReturnType<typeof setInterval>

    async function load() {
      try {
        const [h, m] = await Promise.all([fetchHealth(), fetchMetrics()])
        setHealth(h)
        setMetrics(m)
        setError(null)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load')
      }
    }

    load()
    timer = setInterval(load, 30000)
    return () => clearInterval(timer)
  }, [])

  function formatUptime(s: number): string {
    const d = Math.floor(s / 86400)
    const h = Math.floor((s % 86400) / 3600)
    const m = Math.floor((s % 3600) / 60)
    if (d > 0) return `${d}d ${h}h`
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
  }

  const statusColor = {
    healthy: 'text-emerald-400',
    degraded: 'text-yellow-400',
    unhealthy: 'text-red-400',
  }

  return (
    <div className="p-4 space-y-4 overflow-y-auto h-full">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">System Health</h2>

      {error && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300">
          {error}
        </div>
      )}

      {health && (
        <div className="space-y-3">
          <div className="flex items-center gap-3 bg-gray-800/50 rounded-lg p-3">
            <Server className="w-5 h-5 text-gray-400" />
            <div>
              <p className={`text-sm font-medium ${statusColor[health.status]}`}>
                {health.status.toUpperCase()}
              </p>
              <p className="text-xs text-gray-500">Server Status</p>
            </div>
          </div>

          <div className="flex items-center gap-3 bg-gray-800/50 rounded-lg p-3">
            <Clock className="w-5 h-5 text-gray-400" />
            <div>
              <p className="text-sm font-medium text-gray-200">{formatUptime(health.uptime_seconds)}</p>
              <p className="text-xs text-gray-500">Uptime</p>
            </div>
          </div>

          {metrics && (
            <div className="flex items-center gap-3 bg-gray-800/50 rounded-lg p-3">
              <Activity className="w-5 h-5 text-gray-400" />
              <div>
                <p className="text-sm font-medium text-gray-200">
                  {metrics.events_processed_total.toLocaleString()}
                </p>
                <p className="text-xs text-gray-500">Events Processed</p>
              </div>
            </div>
          )}

          {health.checks && Object.entries(health.checks).map(([name, check]) => (
            <div key={name} className="bg-gray-800/50 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-300 capitalize">{name}</span>
                <span className={`text-xs font-medium ${statusColor[check.status] || 'text-gray-400'}`}>
                  {check.status}
                </span>
              </div>
              {check.message && (
                <p className="text-xs text-gray-500 mt-1">{check.message}</p>
              )}
            </div>
          ))}

          {metrics?.events_by_provider && Object.keys(metrics.events_by_provider).length > 0 && (
            <>
              <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mt-4">Events by Provider</h3>
              {Object.entries(metrics.events_by_provider)
                .sort(([, a], [, b]) => b - a)
                .map(([provider, count]) => (
                  <div key={provider} className="flex items-center justify-between bg-gray-800/30 rounded px-3 py-2">
                    <span className="text-xs text-gray-400">{provider}</span>
                    <span className="text-xs font-mono text-gray-300">{count.toLocaleString()}</span>
                  </div>
                ))
              }
            </>
          )}
        </div>
      )}

      {!health && !error && (
        <p className="text-sm text-gray-500">Loading health data...</p>
      )}
    </div>
  )
}
