import { useState } from 'react'
import type { EventFilters } from '../../types/sentinel'
import { Search, Filter, X, Satellite } from 'lucide-react'

const KNOWN_CATEGORIES = [
  'earthquake', 'weather', 'wildfire', 'flood', 'volcano', 'tsunami',
  'conflict', 'aviation', 'maritime', 'health', 'satellite', 'space_weather',
  'financial', 'piracy', 'disaster',
]

interface Props {
  filters: EventFilters
  onChange: (filters: EventFilters) => void
  sources: string[]
  categories: string[]
}

export function FilterPanel({ filters, onChange, sources, categories }: Props) {
  const [expanded, setExpanded] = useState(true)

  function update(partial: Partial<EventFilters>) {
    onChange({ ...filters, ...partial })
  }

  function clear() {
    onChange({})
  }

  const allCategories = Array.from(new Set([...KNOWN_CATEGORIES, ...categories])).sort()
  const hidingSatellites = filters.exclude_category?.includes('satellite')

  const hasFilters = filters.source || filters.category || filters.severity || filters.q || filters.min_magnitude || filters.exclude_category

  return (
    <div className="border-b border-gray-800 bg-gray-900/50">
      <div className="px-4 py-2 flex items-center gap-2">
        <Search className="w-4 h-4 text-gray-500" />
        <input
          type="text"
          value={filters.q || ''}
          onChange={e => update({ q: e.target.value || undefined })}
          placeholder="Search events..."
          className="flex-1 bg-transparent text-sm text-gray-200 placeholder-gray-600 outline-none"
        />
        <button
          onClick={() => setExpanded(!expanded)}
          className={`p-1.5 rounded ${expanded ? 'bg-gray-700' : 'hover:bg-gray-800'} transition-colors`}
        >
          <Filter className="w-4 h-4 text-gray-400" />
        </button>
        {hasFilters && (
          <button onClick={clear} className="p-1.5 rounded hover:bg-gray-800 transition-colors">
            <X className="w-4 h-4 text-gray-400" />
          </button>
        )}
      </div>

      {expanded && (
        <div className="px-4 pb-3 space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <select
              value={filters.severity || ''}
              onChange={e => update({ severity: e.target.value || undefined })}
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300"
            >
              <option value="">All Severities</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>

            <select
              value={filters.source || ''}
              onChange={e => update({ source: e.target.value || undefined })}
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300"
            >
              <option value="">All Sources</option>
              {sources.map(s => <option key={s} value={s}>{s}</option>)}
            </select>

            <select
              value={filters.category || ''}
              onChange={e => update({ category: e.target.value || undefined })}
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300"
            >
              <option value="">All Categories</option>
              {allCategories.map(c => <option key={c} value={c}>{c}</option>)}
            </select>

            <input
              type="number"
              value={filters.min_magnitude ?? ''}
              onChange={e => update({ min_magnitude: e.target.value ? Number(e.target.value) : undefined })}
              placeholder="Min magnitude"
              step="0.1"
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 placeholder-gray-600"
            />
          </div>

          {/* Quick exclude toggles */}
          <div className="flex flex-wrap gap-1.5 pt-1">
            <button
              onClick={() => {
                if (hidingSatellites) {
                  update({ exclude_category: undefined })
                } else {
                  update({ exclude_category: 'satellite' })
                }
              }}
              className={`flex items-center gap-1 text-xs px-2 py-1 rounded border transition-colors ${
                hidingSatellites
                  ? 'bg-amber-900/40 border-amber-700 text-amber-300'
                  : 'bg-gray-800 border-gray-700 text-gray-500 hover:text-gray-300'
              }`}
            >
              <Satellite className="w-3 h-3" />
              {hidingSatellites ? 'Satellites hidden' : 'Hide satellites'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
