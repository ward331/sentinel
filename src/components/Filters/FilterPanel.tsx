import { useState, useMemo } from 'react'
import type { EventFilters } from '../../types/sentinel'
import { Search, Filter, X, Satellite, ChevronDown, ChevronRight, Clock } from 'lucide-react'

// ─── Category group definitions ──────────────────────────────────────
interface CategoryGroup {
  label: string
  color: string
  members: string[]
}

const CATEGORY_GROUPS: CategoryGroup[] = [
  { label: 'Geophysical',       color: '#ef4444', members: ['earthquake', 'volcano', 'tsunami'] },
  { label: 'Weather',           color: '#3b82f6', members: ['weather', 'thunderstorm', 'winter_storm', 'wind', 'tornado', 'flood', 'heat_wave', 'drought', 'storm'] },
  { label: 'Hazards',           color: '#f97316', members: ['wildfire', 'disaster'] },
  { label: 'Military/Security', color: '#b91c1c', members: ['conflict', 'troop_movement', 'missile', 'security', 'piracy'] },
  { label: 'Aviation/Maritime', color: '#2563eb', members: ['aviation', 'flight', 'maritime'] },
  { label: 'OSINT/Intel',       color: '#a855f7', members: ['osint', 'gdelt', 'news'] },
  { label: 'Space',             color: '#6b7280', members: ['satellite', 'space_weather'] },
  { label: 'Financial',         color: '#10b981', members: ['financial'] },
  { label: 'Health',            color: '#ec4899', members: ['health', 'epidemic'] },
]

// Build a lookup: category -> group color
const CATEGORY_TO_GROUP_COLOR: Record<string, string> = {}
for (const g of CATEGORY_GROUPS) {
  for (const m of g.members) {
    CATEGORY_TO_GROUP_COLOR[m] = g.color
  }
}

export { CATEGORY_TO_GROUP_COLOR }

interface Props {
  filters: EventFilters
  onChange: (filters: EventFilters) => void
  sources: string[]
  categories: string[]
  selectedCategories: Set<string>
  onToggleCategory: (cat: string) => void
  onClearCategories: () => void
  onSelectAllCategories: () => void
  eventCounts?: Record<string, number>
}

const TIME_PRESETS = [
  { label: '1h', hours: 1 },
  { label: '6h', hours: 6 },
  { label: '24h', hours: 24 },
  { label: '7d', hours: 168 },
  { label: 'All', hours: 0 },
] as const

export function FilterPanel({ filters, onChange, sources, categories, selectedCategories, onToggleCategory, onClearCategories, onSelectAllCategories, eventCounts = {} }: Props) {
  const [expanded, setExpanded] = useState(true)
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())

  function update(partial: Partial<EventFilters>) {
    onChange({ ...filters, ...partial })
  }

  const hidingSatellites = filters.exclude_category?.includes('satellite')
  const hasFilters = filters.source || filters.severity || filters.q || filters.min_magnitude || filters.exclude_category || selectedCategories.size > 0 || filters.start_time

  // Count active filters for summary
  const activeFilterCount = [
    filters.source,
    filters.severity,
    filters.q,
    filters.min_magnitude,
    filters.exclude_category,
    filters.start_time,
    selectedCategories.size > 0 ? true : undefined,
  ].filter(Boolean).length

  // Determine active time preset
  function getActivePreset(): string | null {
    if (!filters.start_time) return 'All'
    const start = new Date(filters.start_time).getTime()
    const now = Date.now()
    const diffHours = (now - start) / (1000 * 60 * 60)
    for (const p of TIME_PRESETS) {
      if (p.hours === 0) continue
      if (Math.abs(diffHours - p.hours) < 0.5) return p.label
    }
    return null
  }

  function setTimePreset(hours: number) {
    if (hours === 0) {
      update({ start_time: undefined })
    } else {
      const start = new Date(Date.now() - hours * 60 * 60 * 1000).toISOString()
      update({ start_time: start })
    }
  }

  // Build groups with only categories that actually exist in data, plus an "Other" bucket
  const { groups, uncategorized } = useMemo(() => {
    const catSet = new Set(categories)
    const assigned = new Set<string>()

    const groups = CATEGORY_GROUPS.map(g => {
      const present = g.members.filter(m => catSet.has(m))
      for (const m of present) assigned.add(m)
      return { ...g, present }
    }).filter(g => g.present.length > 0)

    const uncategorized = categories.filter(c => !assigned.has(c))
    return { groups, uncategorized }
  }, [categories])

  function toggleGroup(label: string) {
    setCollapsedGroups(prev => {
      const next = new Set(prev)
      if (next.has(label)) next.delete(label)
      else next.add(label)
      return next
    })
  }

  function selectGroupAll(members: string[]) {
    // Add all members of this group to selected
    const present = members.filter(m => categories.includes(m))
    for (const m of present) {
      if (!selectedCategories.has(m)) {
        onToggleCategory(m)
      }
    }
  }

  function selectGroupNone(members: string[]) {
    // Remove all members of this group from selected
    const present = members.filter(m => categories.includes(m))
    for (const m of present) {
      if (selectedCategories.has(m)) {
        onToggleCategory(m)
      }
    }
  }

  function groupActiveCount(members: string[]): number {
    return members.filter(m => selectedCategories.has(m)).length
  }

  function groupPresentCount(members: string[]): number {
    return members.filter(m => categories.includes(m)).length
  }

  return (
    <div className="border-b border-gray-800 bg-gray-900/50">
      {/* Search bar */}
      <div className="px-3 py-2 flex items-center gap-2">
        <Search className="w-4 h-4 text-gray-500 shrink-0" />
        <input
          type="text"
          value={filters.q || ''}
          onChange={e => update({ q: e.target.value || undefined })}
          placeholder="Search events..."
          className="flex-1 bg-transparent text-sm text-gray-200 placeholder-gray-600 outline-none min-w-0"
        />
        <button
          onClick={() => setExpanded(!expanded)}
          className={`p-1.5 rounded shrink-0 ${expanded ? 'bg-gray-700' : 'hover:bg-gray-800'} transition-colors`}
        >
          <Filter className="w-4 h-4 text-gray-400" />
        </button>
        {hasFilters && (
          <button onClick={() => { onChange({ exclude_category: 'satellite' }); onClearCategories() }}
                  className="p-1.5 rounded hover:bg-gray-800 transition-colors shrink-0">
            <X className="w-4 h-4 text-gray-400" />
          </button>
        )}
      </div>

      {/* Active filter summary when collapsed */}
      {!expanded && activeFilterCount > 0 && (
        <div className="px-3 pb-2">
          <span className="text-[11px] text-gray-500">
            {activeFilterCount} filter{activeFilterCount !== 1 ? 's' : ''} active
          </span>
        </div>
      )}

      {expanded && (
        <div className="px-3 pb-3 space-y-2">
          {/* Severity + Source + Magnitude row */}
          <div className="flex gap-2">
            <select
              value={filters.severity || ''}
              onChange={e => update({ severity: e.target.value || undefined })}
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 flex-1"
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
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 flex-1"
            >
              <option value="">All Sources</option>
              {sources.map(s => <option key={s} value={s}>{s}</option>)}
            </select>

            <input
              type="number"
              value={filters.min_magnitude ?? ''}
              onChange={e => update({ min_magnitude: e.target.value ? Number(e.target.value) : undefined })}
              placeholder="Min mag"
              step="0.1"
              className="bg-gray-800 border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 placeholder-gray-600 w-20"
            />
          </div>

          {/* Time range presets */}
          <div className="flex items-center gap-2">
            <Clock className="w-3.5 h-3.5 text-gray-500 shrink-0" />
            <div className="flex gap-1 flex-1">
              {TIME_PRESETS.map(p => {
                const isActive = getActivePreset() === p.label
                return (
                  <button
                    key={p.label}
                    onClick={() => setTimePreset(p.hours)}
                    className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-600/30 text-emerald-400 border border-emerald-700'
                        : 'bg-gray-800 text-gray-500 border border-gray-700 hover:text-gray-300 hover:border-gray-600'
                    }`}
                  >
                    {p.label}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Divider */}
          <div className="border-t border-gray-800" />

          {/* Category groups header with global All/None */}
          <div className="flex items-center justify-between">
            <span className="text-[10px] uppercase tracking-wider text-gray-500 font-medium">Categories</span>
            <div className="flex gap-2">
              <button onClick={onSelectAllCategories}
                      className="text-[10px] text-gray-500 hover:text-gray-300 transition-colors">
                All
              </button>
              <button onClick={onClearCategories}
                      className="text-[10px] text-gray-500 hover:text-gray-300 transition-colors">
                None
              </button>
            </div>
          </div>

          {/* Grouped category sections */}
          <div className="space-y-0.5 max-h-[45vh] overflow-y-auto pr-0.5 -mr-0.5">
            {groups.map(group => {
              const isCollapsed = collapsedGroups.has(group.label)
              const activeCount = groupActiveCount(group.present)
              const presentCount = groupPresentCount(group.members)

              return (
                <div key={group.label} className="rounded bg-gray-800/30">
                  {/* Group header */}
                  <div className="flex items-center gap-1.5 px-2 py-1 cursor-pointer select-none"
                       onClick={() => toggleGroup(group.label)}>
                    {isCollapsed
                      ? <ChevronRight className="w-3 h-3 text-gray-500 shrink-0" />
                      : <ChevronDown className="w-3 h-3 text-gray-500 shrink-0" />
                    }
                    <span className="w-2 h-2 rounded-full shrink-0" style={{ background: group.color }} />
                    <span className="text-[11px] font-medium text-gray-300 flex-1">{group.label}</span>
                    {activeCount > 0 && (
                      <span className="text-[9px] px-1.5 py-0.5 rounded-full font-medium"
                            style={{ background: `${group.color}33`, color: group.color }}>
                        {activeCount}/{presentCount}
                      </span>
                    )}
                    <button
                      onClick={e => { e.stopPropagation(); selectGroupAll(group.members) }}
                      className="text-[9px] text-gray-500 hover:text-gray-300 transition-colors px-0.5"
                    >
                      all
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); selectGroupNone(group.members) }}
                      className="text-[9px] text-gray-500 hover:text-gray-300 transition-colors px-0.5"
                    >
                      none
                    </button>
                  </div>

                  {/* Category chips */}
                  {!isCollapsed && (
                    <div className="flex flex-wrap gap-1 px-2 pb-1.5 pt-0.5">
                      {group.present.map(cat => {
                        const active = selectedCategories.has(cat)
                        return (
                          <button
                            key={cat}
                            onClick={() => onToggleCategory(cat)}
                            className="text-[11px] px-2 py-0.5 rounded-full border transition-all flex items-center gap-1"
                            style={{
                              background: active ? `${group.color}22` : 'transparent',
                              borderColor: active ? group.color : '#374151',
                              color: active ? group.color : '#6b7280',
                              fontWeight: active ? 600 : 400,
                            }}
                          >
                            {cat.replace(/_/g, ' ')}
                            {eventCounts[cat] !== undefined && (
                              <span className="text-[9px] opacity-60">{eventCounts[cat]}</span>
                            )}
                          </button>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })}

            {/* Uncategorized bucket */}
            {uncategorized.length > 0 && (
              <div className="rounded bg-gray-800/30">
                <div className="flex items-center gap-1.5 px-2 py-1 cursor-pointer select-none"
                     onClick={() => toggleGroup('__other')}>
                  {collapsedGroups.has('__other')
                    ? <ChevronRight className="w-3 h-3 text-gray-500 shrink-0" />
                    : <ChevronDown className="w-3 h-3 text-gray-500 shrink-0" />
                  }
                  <span className="w-2 h-2 rounded-full shrink-0 bg-gray-500" />
                  <span className="text-[11px] font-medium text-gray-300 flex-1">Other</span>
                  <button
                    onClick={e => { e.stopPropagation(); selectGroupAll(uncategorized) }}
                    className="text-[9px] text-gray-500 hover:text-gray-300 transition-colors px-0.5"
                  >
                    all
                  </button>
                  <button
                    onClick={e => { e.stopPropagation(); selectGroupNone(uncategorized) }}
                    className="text-[9px] text-gray-500 hover:text-gray-300 transition-colors px-0.5"
                  >
                    none
                  </button>
                </div>
                {!collapsedGroups.has('__other') && (
                  <div className="flex flex-wrap gap-1 px-2 pb-1.5 pt-0.5">
                    {uncategorized.map(cat => {
                      const active = selectedCategories.has(cat)
                      return (
                        <button
                          key={cat}
                          onClick={() => onToggleCategory(cat)}
                          className="text-[11px] px-2 py-0.5 rounded-full border transition-all flex items-center gap-1"
                          style={{
                            background: active ? '#6b728022' : 'transparent',
                            borderColor: active ? '#6b7280' : '#374151',
                            color: active ? '#9ca3af' : '#6b7280',
                            fontWeight: active ? 600 : 400,
                          }}
                        >
                          {cat.replace(/_/g, ' ')}
                          {eventCounts[cat] !== undefined && (
                            <span className="text-[9px] opacity-60">{eventCounts[cat]}</span>
                          )}
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Divider */}
          <div className="border-t border-gray-800" />

          {/* Satellite toggle */}
          <div className="flex gap-1.5 pt-0.5">
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
