import { useState, useEffect, useCallback } from 'react'
import { fetchOsintResources, fetchOsintCategories, fetchOsintPlatforms } from '../../api/client'
import type { OSINTResource } from '../../types/sentinel'
import { Search, ExternalLink, Key, Filter, AlertTriangle, X, ChevronDown } from 'lucide-react'

const PLATFORM_COLORS: Record<string, string> = {
  web: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
  api: 'bg-purple-500/20 text-purple-300 border-purple-500/30',
  dataset: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
  tool: 'bg-orange-500/20 text-orange-300 border-orange-500/30',
  rss: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
  map: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30',
}

const CREDIBILITY_COLORS: Record<string, string> = {
  verified_osint: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
  official: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
  community: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
  experimental: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
}

const PAGE_SIZE = 24

export function OsintBrowser() {
  const [resources, setResources] = useState<OSINTResource[]>([])
  const [total, setTotal] = useState(0)
  const [categories, setCategories] = useState<string[]>([])
  const [platforms, setPlatforms] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [searchText, setSearchText] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('')
  const [selectedPlatform, setSelectedPlatform] = useState('')
  const [freeTierOnly, setFreeTierOnly] = useState(false)
  const [builtinOnly, setBuiltinOnly] = useState(false)
  const [offset, setOffset] = useState(0)
  const [showFilters, setShowFilters] = useState(true)

  // Load categories and platforms on mount
  useEffect(() => {
    async function loadMeta() {
      try {
        const [cats, plats] = await Promise.all([fetchOsintCategories(), fetchOsintPlatforms()])
        setCategories(cats)
        setPlatforms(plats)
      } catch {
        // non-critical, filters just won't populate
      }
    }
    loadMeta()
  }, [])

  const loadResources = useCallback(async (resetOffset = false) => {
    setLoading(true)
    try {
      const currentOffset = resetOffset ? 0 : offset
      if (resetOffset) setOffset(0)

      const params: Record<string, string> = {
        limit: String(PAGE_SIZE),
        offset: String(currentOffset),
      }
      if (searchText.trim()) params.q = searchText.trim()
      if (selectedCategory) params.category = selectedCategory
      if (selectedPlatform) params.platform = selectedPlatform
      if (freeTierOnly) params.free_tier = 'true'
      if (builtinOnly) params.is_builtin = 'true'

      const result = await fetchOsintResources(params)

      if (resetOffset || currentOffset === 0) {
        setResources(result.resources)
      } else {
        setResources(prev => [...prev, ...result.resources])
      }
      setTotal(result.total)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load resources')
    } finally {
      setLoading(false)
    }
  }, [searchText, selectedCategory, selectedPlatform, freeTierOnly, builtinOnly, offset])

  // Reload when filters change (reset offset)
  useEffect(() => {
    loadResources(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchText, selectedCategory, selectedPlatform, freeTierOnly, builtinOnly])

  // Load more (uses current offset)
  function handleLoadMore() {
    const newOffset = offset + PAGE_SIZE
    setOffset(newOffset)
  }

  // When offset changes (and is non-zero), fetch next page
  useEffect(() => {
    if (offset > 0) loadResources(false)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [offset])

  const hasMore = resources.length < total

  const activeFilterCount = [
    searchText.trim(),
    selectedCategory,
    selectedPlatform,
    freeTierOnly,
    builtinOnly,
  ].filter(Boolean).length

  function clearFilters() {
    setSearchText('')
    setSelectedCategory('')
    setSelectedPlatform('')
    setFreeTierOnly(false)
    setBuiltinOnly(false)
  }

  return (
    <div className="p-6 space-y-4 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">
          OSINT Resources
        </h2>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-500">{total} resources</span>
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded text-xs font-medium transition-colors ${
              showFilters
                ? 'bg-emerald-500/20 text-emerald-300'
                : 'bg-gray-800 text-gray-400 hover:text-gray-300'
            }`}
          >
            <Filter className="w-3.5 h-3.5" />
            Filters
            {activeFilterCount > 0 && (
              <span className="bg-emerald-400/30 text-emerald-300 rounded-full px-1.5 text-[10px]">
                {activeFilterCount}
              </span>
            )}
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Filter sidebar / panel */}
      {showFilters && (
        <div className="bg-gray-900 border border-gray-700/50 rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Filters</span>
            {activeFilterCount > 0 && (
              <button
                onClick={clearFilters}
                className="text-xs text-gray-500 hover:text-gray-300 flex items-center gap-1"
              >
                <X className="w-3 h-3" />
                Clear all
              </button>
            )}
          </div>

          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
            <input
              type="text"
              placeholder="Search resources..."
              value={searchText}
              onChange={e => setSearchText(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg pl-9 pr-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:outline-none focus:border-emerald-500/50"
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {/* Category */}
            <div className="relative">
              <select
                value={selectedCategory}
                onChange={e => setSelectedCategory(e.target.value)}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 appearance-none focus:outline-none focus:border-emerald-500/50"
              >
                <option value="">All Categories</option>
                {categories.map(c => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
              <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 pointer-events-none" />
            </div>

            {/* Platform */}
            <div className="relative">
              <select
                value={selectedPlatform}
                onChange={e => setSelectedPlatform(e.target.value)}
                className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 appearance-none focus:outline-none focus:border-emerald-500/50"
              >
                <option value="">All Platforms</option>
                {platforms.map(p => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
              <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500 pointer-events-none" />
            </div>

            {/* Checkboxes */}
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={freeTierOnly}
                onChange={e => setFreeTierOnly(e.target.checked)}
                className="rounded border-gray-600 bg-gray-800 text-emerald-500 focus:ring-emerald-500/30"
              />
              <span className="text-sm text-gray-300">Free tier only</span>
            </label>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={builtinOnly}
                onChange={e => setBuiltinOnly(e.target.checked)}
                className="rounded border-gray-600 bg-gray-800 text-emerald-500 focus:ring-emerald-500/30"
              />
              <span className="text-sm text-gray-300">Built-in only</span>
            </label>
          </div>
        </div>
      )}

      {/* Resource grid */}
      {loading && resources.length === 0 ? (
        <p className="text-sm text-gray-500">Loading resources...</p>
      ) : resources.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-sm text-gray-500">No resources found matching your filters.</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            {resources.map(resource => (
              <ResourceCard key={resource.id} resource={resource} />
            ))}
          </div>

          {hasMore && (
            <div className="flex justify-center pt-2">
              <button
                onClick={handleLoadMore}
                disabled={loading}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-lg text-sm text-gray-300 transition-colors disabled:opacity-50"
              >
                {loading ? 'Loading...' : `Load more (${resources.length} of ${total})`}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function ResourceCard({ resource }: { resource: OSINTResource }) {
  const platformStyle = PLATFORM_COLORS[resource.platform] || 'bg-gray-500/20 text-gray-400 border-gray-500/30'
  const credStyle = CREDIBILITY_COLORS[resource.credibility] || 'bg-gray-500/20 text-gray-400 border-gray-500/30'

  return (
    <div className="bg-gray-800 rounded-lg border border-gray-700/50 p-4 flex flex-col gap-2.5 hover:border-gray-600/50 transition-colors">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-bold text-gray-100 leading-tight">{resource.display_name}</h3>
        <div className="flex items-center gap-1.5 shrink-0">
          {resource.free_tier && (
            <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
              FREE
            </span>
          )}
          {resource.api_key_required && (
            <span title="API key required" className="text-yellow-400">
              <Key className="w-3.5 h-3.5" />
            </span>
          )}
        </div>
      </div>

      {/* Description */}
      {resource.description && (
        <p className="text-xs text-gray-400 leading-relaxed line-clamp-2">{resource.description}</p>
      )}

      {/* Badges */}
      <div className="flex flex-wrap gap-1.5">
        <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded border ${platformStyle}`}>
          {resource.platform}
        </span>
        <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded border ${credStyle}`}>
          {resource.credibility.replace(/_/g, ' ')}
        </span>
        <span className="text-[10px] font-medium px-1.5 py-0.5 rounded border bg-gray-700/50 text-gray-300 border-gray-600/50">
          {resource.category}
        </span>
      </div>

      {/* Tags */}
      {resource.tags && resource.tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {resource.tags.map(tag => (
            <span
              key={tag}
              className="text-[10px] px-1.5 py-0.5 rounded-full bg-gray-700/50 text-gray-500"
            >
              {tag}
            </span>
          ))}
        </div>
      )}

      {/* Link */}
      {resource.profile_url && (
        <a
          href={resource.profile_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-1 text-xs text-emerald-400 hover:text-emerald-300 transition-colors mt-auto"
        >
          <ExternalLink className="w-3 h-3" />
          {(() => {
            try {
              return new URL(resource.profile_url).hostname
            } catch {
              return 'Visit'
            }
          })()}
        </a>
      )}
    </div>
  )
}
