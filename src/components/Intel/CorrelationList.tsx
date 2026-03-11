import { useState, useEffect } from 'react'
import { GitMerge, RefreshCw, AlertTriangle, CheckCircle, MapPin } from 'lucide-react'
import { fetchCorrelations } from '../../api/client'
import type { CorrelationFlash } from '../../types/sentinel'

function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

function intensityColor(eventCount: number): string {
  if (eventCount >= 10) return 'border-l-red-500'
  if (eventCount >= 6) return 'border-l-orange-400'
  if (eventCount >= 3) return 'border-l-yellow-400'
  return 'border-l-emerald-400'
}

function intensityBg(eventCount: number): string {
  if (eventCount >= 10) return 'bg-red-500/10'
  if (eventCount >= 6) return 'bg-orange-400/10'
  if (eventCount >= 3) return 'bg-yellow-400/10'
  return 'bg-emerald-400/5'
}

function countBadgeColor(eventCount: number): string {
  if (eventCount >= 10) return 'bg-red-500/20 text-red-400'
  if (eventCount >= 6) return 'bg-orange-400/20 text-orange-400'
  if (eventCount >= 3) return 'bg-yellow-400/20 text-yellow-400'
  return 'bg-emerald-400/20 text-emerald-400'
}

export function CorrelationList({ onSelect }: { onSelect?: (c: CorrelationFlash) => void }) {
  const [correlations, setCorrelations] = useState<CorrelationFlash[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = async () => {
    try {
      const res = await fetchCorrelations()
      setCorrelations(res.correlations)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch correlations')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    const interval = setInterval(load, 30000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 flex flex-col">
      <div className="flex items-center justify-between p-4 border-b border-gray-800">
        <div className="flex items-center gap-2">
          <GitMerge className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-gray-100 uppercase tracking-wider">Correlations</h3>
          <span className="text-xs text-gray-500">{correlations.length}</span>
        </div>
        <button
          onClick={load}
          disabled={loading}
          className="p-1.5 text-gray-400 hover:text-gray-200 disabled:opacity-50 transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      <div className="overflow-y-auto flex-1" style={{ maxHeight: '500px' }}>
        {loading && correlations.length === 0 && (
          <p className="text-gray-400 text-sm p-4">Loading...</p>
        )}

        {error && (
          <div className="flex items-center gap-2 text-red-400 text-sm p-4">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {!loading && !error && correlations.length === 0 && (
          <p className="text-gray-500 text-sm p-4 text-center">No active correlations.</p>
        )}

        {correlations.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelect?.(c)}
            className={`w-full text-left px-4 py-3 border-l-3 ${intensityColor(c.event_count)} border-b border-b-gray-800/50 hover:bg-gray-800/50 transition-colors ${intensityBg(c.event_count)}`}
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <MapPin className="w-3.5 h-3.5 text-gray-400 shrink-0" />
                  <span className="text-sm font-medium text-gray-200 truncate">
                    {c.incident_name || c.region_name}
                  </span>
                  {c.confirmed && (
                    <CheckCircle className="w-3.5 h-3.5 text-emerald-400 shrink-0" aria-label="Confirmed" />
                  )}
                </div>
                <div className="flex items-center gap-3 mt-1.5 text-xs text-gray-500">
                  <span className={`font-mono px-1.5 py-0.5 rounded ${countBadgeColor(c.event_count)}`}>
                    {c.event_count} events
                  </span>
                  <span>{c.source_count} sources</span>
                  <span>{c.radius_km.toFixed(0)} km</span>
                </div>
              </div>
              <div className="text-right shrink-0">
                <div className="text-xs text-gray-500">{timeAgo(c.started_at)}</div>
                <div className="text-xs text-gray-600 mt-0.5">to {timeAgo(c.last_event_at)}</div>
              </div>
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}
