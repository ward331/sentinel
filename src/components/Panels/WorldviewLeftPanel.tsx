import { useState } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Wifi,
  WifiOff,
  Plane,
  Ship,
  Satellite,
  Radio,
  Flame,
  AlertTriangle,
  Globe,
  Zap,
  Eye,
  EyeOff,
  Server,
  Camera,
} from 'lucide-react'

export type MapStyleKey = 'default' | 'satellite' | 'flir' | 'nvg' | 'crt'

interface WorldviewLeftPanelProps {
  visibleLayers: Set<string>
  onToggleLayer: (layer: string) => void
  mapStyle: MapStyleKey
  onSetMapStyle: (style: MapStyleKey) => void
  freshness: Record<string, string>
  sourceCounts: Record<string, number>
  isConnected: boolean
}

const LAYERS: { key: string; label: string; color: string; icon: React.ReactNode }[] = [
  { key: 'events', label: 'EVENTS', color: '#f59e0b', icon: <AlertTriangle size={12} /> },
  { key: 'aircraft', label: 'AIRCRAFT', color: '#3b82f6', icon: <Plane size={12} /> },
  { key: 'ships', label: 'SHIPS', color: '#06b6d4', icon: <Ship size={12} /> },
  { key: 'satellites', label: 'SATELLITES', color: '#a855f7', icon: <Satellite size={12} /> },
  { key: 'earthquakes', label: 'EARTHQUAKES', color: '#ef4444', icon: <Globe size={12} /> },
  { key: 'fires', label: 'FIRES', color: '#f97316', icon: <Flame size={12} /> },
  { key: 'conflicts', label: 'CONFLICTS', color: '#dc2626', icon: <Zap size={12} /> },
  { key: 'sigint', label: 'SIGINT', color: '#10b981', icon: <Radio size={12} /> },
  { key: 'datacenters', label: 'DATACENTERS', color: '#6366f1', icon: <Server size={12} /> },
  { key: 'cctv', label: 'CCTV', color: '#78716c', icon: <Camera size={12} /> },
]

const MAP_STYLES: { key: MapStyleKey; label: string }[] = [
  { key: 'default', label: 'DEFAULT' },
  { key: 'satellite', label: 'SATELLITE' },
  { key: 'flir', label: 'FLIR' },
  { key: 'nvg', label: 'NVG' },
  { key: 'crt', label: 'CRT' },
]

function freshnessColor(ts: string): string {
  if (!ts) return '#ef4444'
  const ago = Date.now() - new Date(ts).getTime()
  if (ago < 5 * 60_000) return '#22c55e'
  if (ago < 30 * 60_000) return '#eab308'
  return '#ef4444'
}

function relativeTime(ts: string): string {
  if (!ts) return 'N/A'
  const ago = Date.now() - new Date(ts).getTime()
  if (ago < 0) return 'now'
  const sec = Math.floor(ago / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  return `${Math.floor(hr / 24)}d ago`
}

export default function WorldviewLeftPanel({
  visibleLayers,
  onToggleLayer,
  mapStyle,
  onSetMapStyle,
  freshness,
  sourceCounts,
  isConnected,
}: WorldviewLeftPanelProps) {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div
      className={`absolute top-0 left-0 h-full z-20 flex transition-all duration-200 ${
        collapsed ? 'w-8' : 'w-[300px]'
      }`}
    >
      {/* Toggle button */}
      <div className="relative h-full">
        {!collapsed && (
          <div className="h-full w-[300px] bg-gray-950/95 border-r border-gray-800 overflow-y-auto backdrop-blur-sm">
            {/* Connection badge */}
            <div className="flex items-center gap-2 px-3 py-2 border-b border-gray-800">
              {isConnected ? (
                <>
                  <Wifi size={12} className="text-green-500" />
                  <span className="text-[10px] font-mono uppercase tracking-widest text-green-500">
                    ONLINE
                  </span>
                </>
              ) : (
                <>
                  <WifiOff size={12} className="text-red-500" />
                  <span className="text-[10px] font-mono uppercase tracking-widest text-red-500">
                    OFFLINE
                  </span>
                </>
              )}
            </div>

            {/* Layer toggles */}
            <div className="px-3 py-2 border-b border-gray-800">
              <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-2">
                LAYER CONTROL
              </h3>
              <div className="space-y-1">
                {LAYERS.map((layer) => {
                  const active = visibleLayers.has(layer.key)
                  const count = sourceCounts[layer.key] ?? 0
                  return (
                    <button
                      key={layer.key}
                      onClick={() => onToggleLayer(layer.key)}
                      className="w-full flex items-center gap-2 px-2 py-1 rounded hover:bg-gray-900 transition-colors group"
                    >
                      <div
                        className="w-2 h-2 rounded-full shrink-0"
                        style={{
                          backgroundColor: active ? layer.color : '#374151',
                          boxShadow: active ? `0 0 6px ${layer.color}` : 'none',
                        }}
                      />
                      <span className="text-gray-500 group-hover:text-gray-400">
                        {layer.icon}
                      </span>
                      {active ? (
                        <Eye size={10} className="text-gray-600" />
                      ) : (
                        <EyeOff size={10} className="text-gray-700" />
                      )}
                      <span
                        className={`text-[11px] font-mono flex-1 text-left ${
                          active ? 'text-gray-300' : 'text-gray-600'
                        }`}
                      >
                        {layer.label}
                      </span>
                      {count > 0 && (
                        <span
                          className="text-[9px] font-mono px-1.5 py-0.5 rounded-full"
                          style={{
                            backgroundColor: active ? layer.color + '20' : '#1f293720',
                            color: active ? layer.color : '#6b7280',
                          }}
                        >
                          {count > 9999 ? `${(count / 1000).toFixed(1)}k` : count}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Map Style */}
            <div className="px-3 py-2 border-b border-gray-800">
              <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-2">
                MAP STYLE
              </h3>
              <div className="grid grid-cols-5 gap-1">
                {MAP_STYLES.map((style) => (
                  <button
                    key={style.key}
                    onClick={() => onSetMapStyle(style.key)}
                    className={`text-[9px] font-mono uppercase py-1.5 px-1 rounded border transition-all ${
                      mapStyle === style.key
                        ? 'border-cyan-600 bg-cyan-950/50 text-cyan-400 shadow-[0_0_8px_rgba(6,182,212,0.2)]'
                        : 'border-gray-800 bg-gray-900 text-gray-500 hover:border-gray-700 hover:text-gray-400'
                    }`}
                  >
                    {style.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Source Freshness */}
            <div className="px-3 py-2">
              <h3 className="text-[10px] font-mono uppercase tracking-widest text-gray-500 mb-2">
                SOURCE FRESHNESS
              </h3>
              <div className="space-y-1">
                {Object.entries(freshness).map(([source, ts]) => (
                  <div key={source} className="flex items-center gap-2 px-1">
                    <div
                      className="w-1.5 h-1.5 rounded-full shrink-0"
                      style={{ backgroundColor: freshnessColor(ts) }}
                    />
                    <span className="text-[10px] font-mono text-gray-500 flex-1 truncate uppercase">
                      {source.replace(/_/g, ' ')}
                    </span>
                    <span className="text-[10px] font-mono text-gray-600">
                      {relativeTime(ts)}
                    </span>
                  </div>
                ))}
                {Object.keys(freshness).length === 0 && (
                  <span className="text-[10px] font-mono text-gray-700">NO SOURCES</span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Collapse/Expand toggle */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className={`absolute top-1/2 -translate-y-1/2 ${
            collapsed ? 'left-0' : 'left-[300px]'
          } w-6 h-12 bg-gray-950 border border-gray-800 border-l-0 rounded-r flex items-center justify-center text-gray-500 hover:text-gray-300 hover:bg-gray-900 transition-all z-30`}
        >
          {collapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
        </button>
      </div>
    </div>
  )
}
