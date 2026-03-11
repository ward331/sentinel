import { useState } from 'react'
import { FileText, ChevronDown, ChevronRight, RefreshCw, AlertTriangle, Clock, Activity } from 'lucide-react'
import { fetchIntelBriefing } from '../../api/client'
import type { IntelBriefing as IntelBriefingData } from '../../types/sentinel'

function formatTimestamp(ts: string): string {
  return new Date(ts).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function IntelBriefing() {
  const [data, setData] = useState<IntelBriefingData | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [collapsed, setCollapsed] = useState<Record<number, boolean>>({})

  const generate = async () => {
    setLoading(true)
    setError(null)
    try {
      const briefing = await fetchIntelBriefing()
      setData(briefing)
      setCollapsed({})
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate briefing')
    } finally {
      setLoading(false)
    }
  }

  const toggleSection = (idx: number) => {
    setCollapsed((prev) => ({ ...prev, [idx]: !prev[idx] }))
  }

  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800">
      <div className="flex items-center justify-between p-4 border-b border-gray-800">
        <div className="flex items-center gap-2">
          <FileText className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-gray-100 uppercase tracking-wider">Intel Briefing</h3>
        </div>
        <button
          onClick={generate}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 disabled:opacity-50 disabled:cursor-wait transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          {loading ? 'Generating...' : data ? 'Refresh' : 'Generate Briefing'}
        </button>
      </div>

      <div className="p-4">
        {loading && !data && (
          <div className="flex items-center justify-center py-12 text-gray-400">
            <div className="flex flex-col items-center gap-3">
              <RefreshCw className="w-6 h-6 animate-spin" />
              <p className="text-sm">Generating intelligence briefing...</p>
            </div>
          </div>
        )}

        {error && (
          <div className="flex items-center gap-2 text-red-400 text-sm py-4">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {!loading && !error && !data && (
          <p className="text-gray-500 text-sm py-8 text-center">
            Click "Generate Briefing" to create an intelligence summary of current events.
          </p>
        )}

        {data && (
          <>
            <div className="flex flex-wrap items-center gap-4 mb-4 text-xs text-gray-500">
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                {formatTimestamp(data.generated_at)}
              </span>
              <span className="flex items-center gap-1">
                <Activity className="w-3.5 h-3.5" />
                {data.event_count} events
              </span>
              <span>{data.window_hours}h window</span>
              <span className="uppercase text-gray-600">{data.type}</span>
            </div>

            <div className="space-y-1">
              {data.sections.map((section, idx) => {
                const isCollapsed = collapsed[idx] ?? false
                return (
                  <div key={idx} className="border border-gray-800 rounded">
                    <button
                      onClick={() => toggleSection(idx)}
                      className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-800/50 transition-colors"
                    >
                      {isCollapsed ? (
                        <ChevronRight className="w-4 h-4 text-gray-500 shrink-0" />
                      ) : (
                        <ChevronDown className="w-4 h-4 text-gray-400 shrink-0" />
                      )}
                      <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">
                        {section.title}
                      </span>
                    </button>
                    {!isCollapsed && (
                      <div className="px-3 pb-3">
                        <pre className="text-sm text-gray-300 font-mono whitespace-pre-wrap leading-relaxed bg-gray-950 rounded p-3 overflow-x-auto">
                          {section.content}
                        </pre>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </>
        )}
      </div>
    </div>
  )
}
