import { useState } from 'react'
import { ChevronDown, ChevronUp, Layers } from 'lucide-react'

interface MapLegendProps {
  visibleLayers: Set<string>
}

interface LegendEntry {
  label: string
  color: string
  shape?: 'circle' | 'square' | 'diamond'
}

const LAYER_LEGENDS: Record<string, { title: string; entries: LegendEntry[] }> = {
  aircraft: {
    title: 'AIRCRAFT',
    entries: [
      { label: 'Commercial', color: '#3b82f6', shape: 'circle' },
      { label: 'Military', color: '#ef4444', shape: 'diamond' },
      { label: 'Private', color: '#a855f7', shape: 'circle' },
    ],
  },
  ships: {
    title: 'VESSELS',
    entries: [
      { label: 'Cargo', color: '#06b6d4', shape: 'square' },
      { label: 'Tanker', color: '#f97316', shape: 'square' },
      { label: 'Military', color: '#ef4444', shape: 'diamond' },
      { label: 'Passenger', color: '#22c55e', shape: 'square' },
      { label: 'Fishing', color: '#eab308', shape: 'circle' },
    ],
  },
  events: {
    title: 'SEVERITY',
    entries: [
      { label: 'Critical', color: '#ef4444' },
      { label: 'High', color: '#f97316' },
      { label: 'Medium', color: '#eab308' },
      { label: 'Low', color: '#6b7280' },
    ],
  },
  earthquakes: {
    title: 'MAGNITUDE',
    entries: [
      { label: 'M 7.0+', color: '#ef4444' },
      { label: 'M 5.0-6.9', color: '#f97316' },
      { label: 'M 3.0-4.9', color: '#eab308' },
      { label: 'M < 3.0', color: '#22c55e' },
    ],
  },
  satellites: {
    title: 'SATELLITES',
    entries: [
      { label: 'Military/Recon', color: '#ef4444', shape: 'diamond' },
      { label: 'SIGINT', color: '#10b981', shape: 'diamond' },
      { label: 'Navigation', color: '#3b82f6', shape: 'circle' },
      { label: 'Commercial', color: '#a855f7', shape: 'circle' },
    ],
  },
  fires: {
    title: 'FIRES',
    entries: [
      { label: 'High FRP', color: '#ef4444' },
      { label: 'Medium FRP', color: '#f97316' },
      { label: 'Low FRP', color: '#eab308' },
    ],
  },
  conflicts: {
    title: 'CONFLICTS',
    entries: [
      { label: 'Active conflict', color: '#dc2626' },
      { label: 'Tension zone', color: '#f97316' },
    ],
  },
  sigint: {
    title: 'SIGINT',
    entries: [
      { label: 'KiwiSDR', color: '#10b981', shape: 'circle' },
    ],
  },
  datacenters: {
    title: 'DATACENTERS',
    entries: [
      { label: 'Data center', color: '#6366f1', shape: 'square' },
    ],
  },
}

function ShapeIndicator({ color, shape }: { color: string; shape?: string }) {
  const style = { backgroundColor: color }
  if (shape === 'diamond')
    return (
      <div className="w-2.5 h-2.5 shrink-0 rotate-45" style={style} />
    )
  if (shape === 'square')
    return <div className="w-2.5 h-2.5 shrink-0 rounded-sm" style={style} />
  return <div className="w-2.5 h-2.5 shrink-0 rounded-full" style={style} />
}

export default function MapLegend({ visibleLayers }: MapLegendProps) {
  const [expanded, setExpanded] = useState(false)

  const activeLegends = Object.entries(LAYER_LEGENDS).filter(([key]) =>
    visibleLayers.has(key)
  )

  if (activeLegends.length === 0) return null

  return (
    <div className="absolute bottom-8 left-4 z-20">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1.5 px-2 py-1 bg-gray-950/90 border border-gray-800 rounded text-[10px] font-mono text-gray-500 hover:text-gray-300 hover:border-gray-700 transition-all mb-1"
      >
        <Layers size={10} />
        LEGEND
        {expanded ? <ChevronDown size={10} /> : <ChevronUp size={10} />}
      </button>

      {expanded && (
        <div className="bg-gray-950/90 border border-gray-800 rounded-lg p-2.5 backdrop-blur-sm max-w-xs max-h-72 overflow-y-auto">
          <div className="space-y-2.5">
            {activeLegends.map(([key, legend]) => (
              <div key={key}>
                <h4 className="text-[9px] font-mono uppercase tracking-widest text-gray-500 mb-1">
                  {legend.title}
                </h4>
                <div className="space-y-0.5">
                  {legend.entries.map((entry, i) => (
                    <div key={i} className="flex items-center gap-2 px-1">
                      <ShapeIndicator color={entry.color} shape={entry.shape} />
                      <span className="text-[10px] font-mono text-gray-400">{entry.label}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
