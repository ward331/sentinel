import { useState } from 'react'
import type { EventFilters } from '../../types/sentinel'
import { Search, Filter, X } from 'lucide-react'

interface Props {
  filters: EventFilters
  onChange: (filters: EventFilters) => void
  sources: string[]
  categories: string[]
}

export function FilterPanel({ filters, onChange, sources, categories }: Props) {
  const [expanded, setExpanded] = useState(false)

  function update(partial: Partial<EventFilters>) {
    onChange({ ...filters, ...partial })
  }

  function clear() {
    onChange({})
  }

  const hasFilters = filters.source || filters.category || filters.severity || filters.q || filters.min_magnitude

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
        <div className="px-4 pb-3 grid grid-cols-2 gap-2">
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
            {categories.map(c => <option key={c} value={c}>{c}</option>)}
          </select>

          <div className="flex items-center gap-1">
            <input
              type="number"
              value={filters.min_magnitude ?? ''}
              onChange={e => update({ min_magnitude: e.target.value ? Number(e.target.value) : undefined })}
              placeholder="Min mag"
              step="0.1"
              className="w-full bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 placeholder-gray-600"
            />
          </div>
        </div>
      )}
    </div>
  )
}
