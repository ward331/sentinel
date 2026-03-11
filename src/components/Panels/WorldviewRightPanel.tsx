import { useState, useRef, useEffect, useCallback } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
  Radio,
  Newspaper,
  Zap,
  ExternalLink,
  MapPin,
  Clock,
  Pause,
  Play,
} from 'lucide-react'
import type { SentinelEvent, SignalBoard, CorrelationFlash } from '../../types/sentinel'
import type { KiwiSDR } from '../../types/livedata'

type TabKey = 'events' | 'intel' | 'news' | 'sigint'

interface WorldviewRightPanelProps {
  events: SentinelEvent[]
  onSelectEvent: (e: SentinelEvent) => void
  signalBoard: SignalBoard | null
  correlations: CorrelationFlash[]
  news: Array<{ title: string; link: string; source: string; published: string; summary: string }>
  kiwisdr: KiwiSDR[]
  onFlyTo: (coords: [number, number]) => void
}

const TABS: { key: TabKey; label: string; icon: React.ReactNode }[] = [
  { key: 'events', label: 'EVENTS', icon: <AlertTriangle size={12} /> },
  { key: 'intel', label: 'INTEL', icon: <Zap size={12} /> },
  { key: 'news', label: 'NEWS', icon: <Newspaper size={12} /> },
  { key: 'sigint', label: 'SIGINT', icon: <Radio size={12} /> },
]

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#6b7280',
}

const DOMAIN_COLORS: Record<string, string> = {
  military: '#ef4444',
  cyber: '#a855f7',
  financial: '#eab308',
  natural: '#22c55e',
  health: '#3b82f6',
}

function relativeTime(ts: string): string {
  const ago = Date.now() - new Date(ts).getTime()
  if (ago < 0) return 'now'
  const sec = Math.floor(ago / 1000)
  if (sec < 60) return `${sec}s`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h`
  return `${Math.floor(hr / 24)}d`
}

function ThreatGauge({ domain, value }: { domain: string; value: number }) {
  const color = DOMAIN_COLORS[domain] || '#6b7280'
  const pct = Math.min(Math.max(value, 0), 100)
  return (
    <div className="flex items-center gap-2">
      <span className="text-[10px] font-mono uppercase text-gray-500 w-16 truncate">{domain}</span>
      <div className="flex-1 h-1.5 bg-gray-800 rounded-full overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{ width: `${pct}%`, backgroundColor: color }}
        />
      </div>
      <span className="text-[10px] font-mono w-7 text-right" style={{ color }}>
        {value}
      </span>
    </div>
  )
}

export default function WorldviewRightPanel({
  events,
  onSelectEvent,
  signalBoard,
  correlations,
  news,
  kiwisdr,
  onFlyTo,
}: WorldviewRightPanelProps) {
  const [collapsed, setCollapsed] = useState(false)
  const [activeTab, setActiveTab] = useState<TabKey>('events')
  const [paused, setPaused] = useState(false)
  const feedRef = useRef<HTMLDivElement>(null)

  // Auto-scroll event feed
  useEffect(() => {
    if (activeTab !== 'events' || paused || !feedRef.current) return
    feedRef.current.scrollTop = 0
  }, [events, activeTab, paused])

  const handleEventClick = useCallback(
    (e: SentinelEvent) => {
      onSelectEvent(e)
      if (e.location?.coordinates) {
        const coords = e.location.coordinates as number[]
        if (coords.length >= 2) {
          onFlyTo([coords[0], coords[1]])
        }
      }
    },
    [onSelectEvent, onFlyTo]
  )

  return (
    <div
      className={`absolute top-0 right-0 h-full z-20 flex transition-all duration-200 ${
        collapsed ? 'w-6' : 'w-[380px]'
      }`}
    >
      {/* Collapse toggle */}
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="w-6 h-12 absolute top-1/2 -translate-y-1/2 left-0 -translate-x-full bg-gray-950 border border-gray-800 border-r-0 rounded-l flex items-center justify-center text-gray-500 hover:text-gray-300 hover:bg-gray-900 transition-all z-30"
      >
        {collapsed ? <ChevronLeft size={14} /> : <ChevronRight size={14} />}
      </button>

      {!collapsed && (
        <div className="h-full w-[380px] bg-gray-950/95 border-l border-gray-800 flex flex-col backdrop-blur-sm">
          {/* Tab bar */}
          <div className="flex border-b border-gray-800 shrink-0">
            {TABS.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-[10px] font-mono uppercase tracking-wider transition-all border-b-2 ${
                  activeTab === tab.key
                    ? 'border-cyan-500 text-cyan-400 bg-cyan-950/20'
                    : 'border-transparent text-gray-600 hover:text-gray-400 hover:bg-gray-900/50'
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          <div className="flex-1 overflow-hidden">
            {activeTab === 'events' && (
              <div className="h-full flex flex-col">
                <div className="flex items-center justify-between px-3 py-1.5 border-b border-gray-800/50">
                  <span className="text-[10px] font-mono text-gray-500">
                    {events.length} EVENTS
                  </span>
                  <button
                    onClick={() => setPaused(!paused)}
                    className="text-gray-500 hover:text-gray-300 p-1"
                    title={paused ? 'Resume auto-scroll' : 'Pause auto-scroll'}
                  >
                    {paused ? <Play size={10} /> : <Pause size={10} />}
                  </button>
                </div>
                <div
                  ref={feedRef}
                  className="flex-1 overflow-y-auto"
                  onMouseEnter={() => setPaused(true)}
                  onMouseLeave={() => setPaused(false)}
                >
                  {events.map((evt) => (
                    <button
                      key={evt.id}
                      onClick={() => handleEventClick(evt)}
                      className="w-full text-left flex gap-2 px-3 py-2 border-b border-gray-900 hover:bg-gray-900/60 transition-colors"
                    >
                      {/* Severity stripe */}
                      <div
                        className="w-1 shrink-0 rounded-full self-stretch"
                        style={{ backgroundColor: SEVERITY_COLORS[evt.severity] || '#6b7280' }}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-start justify-between gap-2">
                          <span className="text-[11px] text-gray-300 leading-tight line-clamp-2">
                            {evt.title}
                          </span>
                          <span className="text-[9px] font-mono text-gray-600 shrink-0">
                            {relativeTime(evt.occurred_at)}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 mt-1">
                          <span className="text-[9px] font-mono text-gray-600 uppercase">
                            {evt.source}
                          </span>
                          <span className="text-[9px] font-mono text-gray-700">
                            {evt.category}
                          </span>
                          {evt.magnitude > 0 && (
                            <span className="text-[9px] font-mono text-yellow-600">
                              M{evt.magnitude.toFixed(1)}
                            </span>
                          )}
                        </div>
                      </div>
                    </button>
                  ))}
                  {events.length === 0 && (
                    <div className="px-3 py-8 text-center text-[11px] font-mono text-gray-700">
                      NO EVENTS
                    </div>
                  )}
                </div>
              </div>
            )}

            {activeTab === 'intel' && (
              <div className="h-full overflow-y-auto p-3 space-y-4">
                {/* Signal Board */}
                <div>
                  <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-2">
                    THREAT ASSESSMENT
                  </h3>
                  {signalBoard ? (
                    <div className="space-y-1.5">
                      <ThreatGauge domain="military" value={signalBoard.military} />
                      <ThreatGauge domain="cyber" value={signalBoard.cyber} />
                      <ThreatGauge domain="financial" value={signalBoard.financial} />
                      <ThreatGauge domain="natural" value={signalBoard.natural} />
                      <ThreatGauge domain="health" value={signalBoard.health} />
                      <div className="flex items-center justify-between mt-2 pt-2 border-t border-gray-800/50">
                        <span className="text-[9px] font-mono text-gray-600">
                          {signalBoard.active_alerts ?? 0} ALERTS | {signalBoard.active_correlations ?? 0} CORR
                        </span>
                        <span className="text-[9px] font-mono text-gray-700">
                          {relativeTime(signalBoard.calculated_at)}
                        </span>
                      </div>
                    </div>
                  ) : (
                    <span className="text-[10px] font-mono text-gray-700">NO DATA</span>
                  )}
                </div>

                {/* Correlations */}
                <div>
                  <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-2">
                    CORRELATIONS
                  </h3>
                  <div className="space-y-1.5">
                    {correlations.slice(0, 10).map((corr) => (
                      <button
                        key={corr.id}
                        onClick={() => onFlyTo([corr.lon, corr.lat])}
                        className="w-full text-left px-2 py-1.5 rounded bg-gray-900/50 hover:bg-gray-900 border border-gray-800/50 transition-colors"
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-[11px] text-gray-300 truncate">
                            {corr.incident_name || corr.region_name}
                          </span>
                          {corr.confirmed && (
                            <span className="text-[8px] font-mono bg-red-950 text-red-400 px-1 rounded">
                              CONFIRMED
                            </span>
                          )}
                        </div>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className="text-[9px] font-mono text-gray-600">
                            {corr.event_count} events
                          </span>
                          <span className="text-[9px] font-mono text-gray-700">
                            {corr.source_count} sources
                          </span>
                          <span className="text-[9px] font-mono text-gray-700">
                            {corr.radius_km}km
                          </span>
                        </div>
                      </button>
                    ))}
                    {correlations.length === 0 && (
                      <span className="text-[10px] font-mono text-gray-700">NO CORRELATIONS</span>
                    )}
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'news' && (
              <div className="h-full overflow-y-auto">
                {news.map((item, i) => (
                  <a
                    key={i}
                    href={item.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex gap-2 px-3 py-2 border-b border-gray-900 hover:bg-gray-900/60 transition-colors group"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start gap-2">
                        <span className="text-[11px] text-gray-300 leading-tight line-clamp-2 group-hover:text-cyan-400 transition-colors">
                          {item.title}
                        </span>
                        <ExternalLink
                          size={10}
                          className="text-gray-700 group-hover:text-cyan-600 shrink-0 mt-0.5"
                        />
                      </div>
                      {item.summary && (
                        <p className="text-[10px] text-gray-600 mt-0.5 line-clamp-2">
                          {item.summary}
                        </p>
                      )}
                      <div className="flex items-center gap-2 mt-1">
                        <span className="text-[9px] font-mono px-1 py-0.5 rounded bg-gray-900 text-gray-500 uppercase">
                          {item.source}
                        </span>
                        <span className="text-[9px] font-mono text-gray-700">
                          <Clock size={8} className="inline mr-0.5" />
                          {relativeTime(item.published)}
                        </span>
                      </div>
                    </div>
                  </a>
                ))}
                {news.length === 0 && (
                  <div className="px-3 py-8 text-center text-[11px] font-mono text-gray-700">
                    NO NEWS
                  </div>
                )}
              </div>
            )}

            {activeTab === 'sigint' && (
              <div className="h-full overflow-y-auto p-3 space-y-2">
                <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-1">
                  KIWISDR RECEIVERS ({kiwisdr.length})
                </h3>
                {kiwisdr.map((sdr, i) => (
                  <div
                    key={i}
                    className="px-2 py-2 rounded bg-gray-900/50 border border-gray-800/50"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-[11px] text-gray-300 truncate">{sdr.name}</span>
                      <div className="flex items-center gap-1.5">
                        {sdr.users_active !== undefined && (
                          <span className="text-[9px] font-mono text-gray-600">
                            {sdr.users_active} users
                          </span>
                        )}
                        <button
                          onClick={() => onFlyTo([sdr.lon, sdr.lat])}
                          className="text-gray-600 hover:text-cyan-400 transition-colors"
                          title="Fly to location"
                        >
                          <MapPin size={10} />
                        </button>
                        <a
                          href={sdr.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-gray-600 hover:text-cyan-400 transition-colors"
                          title="Open receiver"
                        >
                          <ExternalLink size={10} />
                        </a>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-[9px] font-mono text-gray-600">
                        {sdr.lat.toFixed(2)}, {sdr.lon.toFixed(2)}
                      </span>
                      {sdr.bands && (
                        <span className="text-[9px] font-mono text-green-700">{sdr.bands}</span>
                      )}
                    </div>
                  </div>
                ))}
                {kiwisdr.length === 0 && (
                  <div className="text-center text-[11px] font-mono text-gray-700 py-8">
                    NO RECEIVERS
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
