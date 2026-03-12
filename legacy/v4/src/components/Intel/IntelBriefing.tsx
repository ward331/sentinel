import { useState } from 'react'
import { FileText, ChevronDown, ChevronRight, RefreshCw, AlertTriangle, Clock, Activity } from 'lucide-react'
import type { SentinelEvent } from '../../types/sentinel'

function formatTimestamp(ts: string): string {
  return new Date(ts).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

interface BriefingSection {
  title: string
  content: string
}

interface IntelBriefingProps {
  events?: SentinelEvent[]
  news?: Array<{ title: string; link: string; source: string; published: string; summary: string }>
}

function generateBriefing(events: SentinelEvent[], news: Array<{ title: string; source: string; summary: string }>): {
  sections: BriefingSection[]
  event_count: number
} {
  const sections: BriefingSection[] = []

  // Events by severity
  const critical = events.filter(e => e.severity === 'critical')
  const high = events.filter(e => e.severity === 'high')

  if (critical.length > 0 || high.length > 0) {
    const lines: string[] = []
    if (critical.length) {
      lines.push(`${critical.length} CRITICAL event(s):`)
      critical.slice(0, 5).forEach(e => lines.push(`  - ${e.title} [${e.source}]`))
    }
    if (high.length) {
      lines.push(`${high.length} HIGH severity event(s):`)
      high.slice(0, 5).forEach(e => lines.push(`  - ${e.title} [${e.source}]`))
    }
    sections.push({ title: 'Priority Events', content: lines.join('\n') })
  }

  // Events by category
  const cats = new Map<string, number>()
  events.forEach(e => cats.set(e.category, (cats.get(e.category) || 0) + 1))
  const sorted = [...cats.entries()].sort((a, b) => b[1] - a[1])
  if (sorted.length > 0) {
    const lines = sorted.map(([cat, count]) => `  ${cat}: ${count} event(s)`)
    sections.push({ title: 'Category Breakdown', content: lines.join('\n') })
  }

  // Events by source
  const sources = new Map<string, number>()
  events.forEach(e => sources.set(e.source, (sources.get(e.source) || 0) + 1))
  const sortedSrc = [...sources.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)
  if (sortedSrc.length > 0) {
    const lines = sortedSrc.map(([src, count]) => `  ${src}: ${count}`)
    sections.push({ title: 'Top Sources', content: lines.join('\n') })
  }

  // News headlines
  if (news.length > 0) {
    const lines = news.slice(0, 10).map(n => `  - [${n.source}] ${n.title}`)
    sections.push({ title: 'Latest Headlines', content: lines.join('\n') })
  }

  if (sections.length === 0) {
    sections.push({ title: 'Status', content: 'No significant events to report.' })
  }

  return { sections, event_count: events.length }
}

export function IntelBriefing({ events = [], news = [] }: IntelBriefingProps) {
  const [briefing, setBriefing] = useState<{ sections: BriefingSection[]; event_count: number; generated_at: string } | null>(null)
  const [loading, setLoading] = useState(false)
  const [collapsed, setCollapsed] = useState<Record<number, boolean>>({})

  const generate = () => {
    setLoading(true)
    // Small delay for UX feedback
    setTimeout(() => {
      const result = generateBriefing(events, news)
      setBriefing({ ...result, generated_at: new Date().toISOString() })
      setCollapsed({})
      setLoading(false)
    }, 300)
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
          {loading ? 'Generating...' : briefing ? 'Refresh' : 'Generate Briefing'}
        </button>
      </div>

      <div className="p-4">
        {loading && !briefing && (
          <div className="flex items-center justify-center py-12 text-gray-400">
            <div className="flex flex-col items-center gap-3">
              <RefreshCw className="w-6 h-6 animate-spin" />
              <p className="text-sm">Generating intelligence briefing...</p>
            </div>
          </div>
        )}

        {!loading && !briefing && (
          <p className="text-gray-500 text-sm py-8 text-center">
            Click "Generate Briefing" to create an intelligence summary of current events.
            {events.length > 0 && (
              <span className="block mt-1 text-gray-600">{events.length} events available for analysis.</span>
            )}
          </p>
        )}

        {briefing && (
          <>
            <div className="flex flex-wrap items-center gap-4 mb-4 text-xs text-gray-500">
              <span className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5" />
                {formatTimestamp(briefing.generated_at)}
              </span>
              <span className="flex items-center gap-1">
                <Activity className="w-3.5 h-3.5" />
                {briefing.event_count} events
              </span>
            </div>

            <div className="space-y-1">
              {briefing.sections.map((section, idx) => {
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
