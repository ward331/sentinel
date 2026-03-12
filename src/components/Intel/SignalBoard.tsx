import { useState, useEffect } from 'react'
import { Shield, Wifi, TrendingUp, Cloud, Heart, AlertTriangle, RefreshCw } from 'lucide-react'
import { fetchSignalBoard } from '../../api/client'
import type { SignalBoard as SignalBoardData } from '../../types/sentinel'

const THREAT_LABELS = ['NOMINAL', 'GUARDED', 'ELEVATED', 'HIGH', 'SEVERE', 'CRITICAL'] as const

const THREAT_COLORS: Record<number, string> = {
  0: 'bg-emerald-400',
  1: 'bg-yellow-400',
  2: 'bg-yellow-400',
  3: 'bg-orange-400',
  4: 'bg-red-400',
  5: 'bg-red-500',
}

const THREAT_TEXT_COLORS: Record<number, string> = {
  0: 'text-emerald-400',
  1: 'text-yellow-400',
  2: 'text-yellow-400',
  3: 'text-orange-400',
  4: 'text-red-400',
  5: 'text-red-500',
}

type Domain = 'military' | 'cyber' | 'financial' | 'natural' | 'health'

const DOMAINS: { key: Domain; label: string; icon: typeof Shield }[] = [
  { key: 'military', label: 'Military', icon: Shield },
  { key: 'cyber', label: 'Cyber', icon: Wifi },
  { key: 'financial', label: 'Financial', icon: TrendingUp },
  { key: 'natural', label: 'Natural', icon: Cloud },
  { key: 'health', label: 'Health', icon: Heart },
]

function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

export function SignalBoard({ initialData }: { initialData?: SignalBoardData | null }) {
  const [data, setData] = useState<SignalBoardData | null>(initialData ?? null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(!initialData)

  // Sync prop changes
  useEffect(() => {
    if (initialData) {
      setData(initialData)
      setLoading(false)
      setError(null)
    }
  }, [initialData])

  const load = async () => {
    try {
      const board = await fetchSignalBoard()
      setData(board)
      setError(null)
    } catch (err) {
      // If we have prop data, silently ignore fetch errors
      if (!data && !initialData) {
        setError(err instanceof Error ? err.message : 'Failed to fetch signal board')
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    // Only self-fetch if no prop data provided
    if (initialData) return
    load()
    const interval = setInterval(load, 30000)
    return () => clearInterval(interval)
  }, [!!initialData])

  if (loading) {
    return (
      <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
        <p className="text-gray-400 text-sm">Loading signal board...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
        <div className="flex items-center gap-2 text-red-400 text-sm">
          <AlertTriangle className="w-4 h-4" />
          <span>{error}</span>
        </div>
      </div>
    )
  }

  if (!data) return null

  const maxLevel = Math.max(data.military, data.cyber, data.financial, data.natural, data.health)

  return (
    <div className="bg-gray-900 rounded-lg p-4 border border-gray-800">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-gray-100 uppercase tracking-wider">Signal Board</h3>
          <span className={`text-xs font-mono px-2 py-0.5 rounded ${THREAT_COLORS[maxLevel]} text-gray-950 font-bold`}>
            {THREAT_LABELS[maxLevel]}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-gray-500">
          {data.active_alerts !== undefined && (
            <span>{data.active_alerts} alerts</span>
          )}
          {data.active_correlations !== undefined && (
            <span>{data.active_correlations} correlations</span>
          )}
          <span>{timeAgo(data.calculated_at)}</span>
          <button onClick={load} className="hover:text-gray-300 transition-colors">
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        {DOMAINS.map(({ key, label, icon: Icon }) => {
          const level = data[key]
          return (
            <div
              key={key}
              className="flex items-center gap-2 bg-gray-800 rounded px-3 py-2 flex-1 min-w-[140px]"
            >
              <Icon className={`w-4 h-4 ${THREAT_TEXT_COLORS[level]} shrink-0`} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-gray-400">{label}</span>
                  <span className={`text-xs font-mono font-bold ${THREAT_TEXT_COLORS[level]}`}>
                    {THREAT_LABELS[level]}
                  </span>
                </div>
                <div className="w-full bg-gray-700 rounded-full h-1.5">
                  <div
                    className={`h-1.5 rounded-full transition-all duration-500 ${THREAT_COLORS[level]}`}
                    style={{ width: `${(level / 5) * 100}%` }}
                  />
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
