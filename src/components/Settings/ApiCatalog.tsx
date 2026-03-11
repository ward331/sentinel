import { useState, useEffect, useMemo } from 'react'
import { Search, ExternalLink, ChevronDown, ChevronRight, X } from 'lucide-react'

interface CatalogApi {
  name: string
  desc: string
  cats: string[]
  free: boolean
  url: string
  type: string
}

interface CatalogData {
  apis: CatalogApi[]
  categories: Record<string, string>
}

// Sentinel-relevant categories shown first
const FEATURED_CATS = [
  'Weather', 'Maps & Geo', 'Transportation', 'Government', 'Security',
  'Space', 'Environment & Nature', 'News & Feeds', 'Health',
  'Finance & Economics', 'Analytics', 'AI & ML',
]

export function ApiCatalog() {
  const [data, setData] = useState<CatalogData | null>(null)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [selectedCat, setSelectedCat] = useState<string | null>(null)
  const [freeOnly, setFreeOnly] = useState(false)
  const [expandedCats, setExpandedCats] = useState<Set<string>>(new Set(FEATURED_CATS.slice(0, 3)))

  useEffect(() => {
    fetch('/api-catalog.json')
      .then(r => r.json())
      .then(d => { setData(d); setLoading(false) })
      .catch(() => setLoading(false))
  }, [])

  const filtered = useMemo(() => {
    if (!data) return []
    let apis = data.apis
    if (selectedCat) apis = apis.filter(a => a.cats.includes(selectedCat))
    if (freeOnly) apis = apis.filter(a => a.free)
    if (search.trim()) {
      const q = search.toLowerCase()
      apis = apis.filter(a =>
        a.name.toLowerCase().includes(q) ||
        a.desc.toLowerCase().includes(q) ||
        a.cats.some(c => c.toLowerCase().includes(q))
      )
    }
    return apis
  }, [data, search, selectedCat, freeOnly])

  // Group filtered results by category
  const grouped = useMemo(() => {
    const map: Record<string, CatalogApi[]> = {}
    for (const api of filtered) {
      const cat = api.cats[0] || 'Other'
      if (!map[cat]) map[cat] = []
      map[cat].push(api)
    }
    // Sort: featured cats first, then alphabetical
    const entries = Object.entries(map)
    entries.sort((a, b) => {
      const ai = FEATURED_CATS.indexOf(a[0])
      const bi = FEATURED_CATS.indexOf(b[0])
      if (ai !== -1 && bi !== -1) return ai - bi
      if (ai !== -1) return -1
      if (bi !== -1) return 1
      return a[0].localeCompare(b[0])
    })
    return entries
  }, [filtered])

  // All unique categories with counts
  const allCats = useMemo(() => {
    if (!data) return []
    const counts: Record<string, number> = {}
    for (const api of data.apis) {
      for (const c of api.cats) {
        counts[c] = (counts[c] || 0) + 1
      }
    }
    return Object.entries(counts).sort((a, b) => {
      const ai = FEATURED_CATS.indexOf(a[0])
      const bi = FEATURED_CATS.indexOf(b[0])
      if (ai !== -1 && bi !== -1) return ai - bi
      if (ai !== -1) return -1
      if (bi !== -1) return 1
      return a[0].localeCompare(b[0])
    })
  }, [data])

  function toggleExpand(cat: string) {
    setExpandedCats(prev => {
      const next = new Set(prev)
      if (next.has(cat)) next.delete(cat)
      else next.add(cat)
      return next
    })
  }

  if (loading) {
    return (
      <div className="text-center text-gray-500 py-12">Loading API catalog...</div>
    )
  }

  if (!data) {
    return (
      <div className="text-center text-gray-500 py-12">
        Failed to load catalog. Place <code className="text-gray-400">api-catalog.json</code> in the public folder.
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Search + filters bar */}
      <div className="flex items-center gap-3">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search 1,288 public APIs..."
            className="w-full bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-8 py-2 text-sm text-gray-300 placeholder-gray-600 outline-none focus:border-gray-500"
          />
          {search && (
            <button onClick={() => setSearch('')} className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
        <label className="flex items-center gap-2 text-xs text-gray-400 cursor-pointer select-none shrink-0">
          <input
            type="checkbox"
            checked={freeOnly}
            onChange={e => setFreeOnly(e.target.checked)}
            className="rounded border-gray-600 bg-gray-800 text-emerald-500"
          />
          Free only
        </label>
      </div>

      {/* Category chips */}
      <div className="flex flex-wrap gap-1.5">
        <button
          onClick={() => setSelectedCat(null)}
          className={`px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
            !selectedCat ? 'bg-emerald-600 text-white' : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
          }`}
        >
          All ({data.apis.length})
        </button>
        {allCats.map(([cat, count]) => {
          const emoji = data.categories[cat] || ''
          const isFeatured = FEATURED_CATS.includes(cat)
          return (
            <button
              key={cat}
              onClick={() => setSelectedCat(selectedCat === cat ? null : cat)}
              className={`px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
                selectedCat === cat
                  ? 'bg-blue-600 text-white'
                  : isFeatured
                    ? 'bg-gray-800 text-gray-300 hover:bg-gray-700 ring-1 ring-gray-700'
                    : 'bg-gray-800/60 text-gray-500 hover:bg-gray-700 hover:text-gray-400'
              }`}
            >
              {emoji} {cat} <span className="opacity-60">{count}</span>
            </button>
          )
        })}
      </div>

      {/* Results count */}
      <div className="text-xs text-gray-500">
        {filtered.length} API{filtered.length !== 1 ? 's' : ''} found
        {selectedCat && <> in <span className="text-gray-400">{selectedCat}</span></>}
        {search && <> matching "<span className="text-gray-400">{search}</span>"</>}
      </div>

      {/* Grouped results */}
      <div className="space-y-1">
        {grouped.map(([cat, apis]) => {
          const emoji = data.categories[cat] || ''
          const isExpanded = expandedCats.has(cat) || !!search || !!selectedCat
          return (
            <div key={cat} className="bg-gray-900 rounded-lg border border-gray-800 overflow-hidden">
              <div
                className="flex items-center gap-2 px-4 py-2.5 cursor-pointer select-none hover:bg-gray-800/50 transition-colors"
                onClick={() => toggleExpand(cat)}
              >
                {isExpanded
                  ? <ChevronDown className="w-4 h-4 text-gray-500 shrink-0" />
                  : <ChevronRight className="w-4 h-4 text-gray-500 shrink-0" />
                }
                <span className="text-sm">{emoji}</span>
                <span className="text-sm font-medium text-gray-300">{cat}</span>
                <span className="text-xs text-gray-600">{apis.length}</span>
              </div>
              {isExpanded && (
                <div className="border-t border-gray-800 divide-y divide-gray-800/50">
                  {apis.slice(0, 50).map(api => (
                    <div key={api.name + api.url} className="px-4 py-2.5 flex items-start gap-3 hover:bg-gray-800/30 transition-colors group">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm text-gray-200 font-medium truncate">{api.name}</span>
                          {api.free && <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-900/40 text-emerald-400 font-medium shrink-0">FREE</span>}
                          {api.type && <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-500 font-mono shrink-0">{api.type}</span>}
                        </div>
                        <div className="text-xs text-gray-500 mt-0.5 line-clamp-2">{api.desc}</div>
                      </div>
                      {api.url && (
                        <a
                          href={api.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="shrink-0 p-1.5 rounded text-gray-600 hover:text-blue-400 hover:bg-gray-800 transition-colors opacity-0 group-hover:opacity-100"
                          title="Open API docs"
                        >
                          <ExternalLink className="w-4 h-4" />
                        </a>
                      )}
                    </div>
                  ))}
                  {apis.length > 50 && (
                    <div className="px-4 py-2 text-xs text-gray-600 text-center">
                      +{apis.length - 50} more — narrow your search to see all
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>

      {filtered.length === 0 && (
        <div className="text-center text-gray-500 py-8">
          No APIs match your search. Try a different keyword.
        </div>
      )}
    </div>
  )
}
