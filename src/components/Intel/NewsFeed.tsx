import { useState, useEffect } from 'react'
import { Newspaper, RefreshCw, AlertTriangle, ExternalLink, Filter } from 'lucide-react'
import { fetchNews } from '../../api/client'
import type { NewsItem } from '../../types/sentinel'

function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

function relevanceColor(score: number): string {
  if (score >= 0.8) return 'bg-emerald-400'
  if (score >= 0.6) return 'bg-yellow-400'
  if (score >= 0.4) return 'bg-orange-400'
  return 'bg-gray-500'
}

function truthColor(score: number): string {
  if (score >= 0.8) return 'text-emerald-400'
  if (score >= 0.6) return 'text-yellow-400'
  if (score >= 0.4) return 'text-orange-400'
  return 'text-red-400'
}

export function NewsFeed() {
  const [items, setItems] = useState<NewsItem[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [categoryFilter, setCategoryFilter] = useState<string>('')

  const load = async () => {
    try {
      setLoading(true)
      const res = await fetchNews(50)
      setItems(res.items)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch news')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const categories = Array.from(new Set(items.map((i) => i.source_category).filter(Boolean))).sort()

  const filtered = categoryFilter
    ? items.filter((i) => i.source_category === categoryFilter)
    : items

  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 flex flex-col">
      <div className="flex items-center justify-between p-4 border-b border-gray-800">
        <div className="flex items-center gap-2">
          <Newspaper className="w-4 h-4 text-emerald-400" />
          <h3 className="text-sm font-semibold text-gray-100 uppercase tracking-wider">News Feed</h3>
          <span className="text-xs text-gray-500">{filtered.length} items</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative flex items-center">
            <Filter className="w-3.5 h-3.5 text-gray-500 absolute left-2 pointer-events-none" />
            <select
              value={categoryFilter}
              onChange={(e) => setCategoryFilter(e.target.value)}
              className="bg-gray-800 text-gray-300 text-xs rounded pl-7 pr-2 py-1.5 border border-gray-700 focus:outline-none focus:border-gray-600 appearance-none cursor-pointer"
            >
              <option value="">All Categories</option>
              {categories.map((cat) => (
                <option key={cat} value={cat}>
                  {cat}
                </option>
              ))}
            </select>
          </div>
          <button
            onClick={load}
            disabled={loading}
            className="p-1.5 text-gray-400 hover:text-gray-200 disabled:opacity-50 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      <div className="overflow-y-auto flex-1" style={{ maxHeight: '600px' }}>
        {loading && items.length === 0 && (
          <p className="text-gray-400 text-sm p-4">Loading...</p>
        )}

        {error && (
          <div className="flex items-center gap-2 text-red-400 text-sm p-4">
            <AlertTriangle className="w-4 h-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {!loading && !error && filtered.length === 0 && (
          <p className="text-gray-500 text-sm p-4 text-center">No news items found.</p>
        )}

        {filtered.map((item) => (
          <div
            key={item.id}
            className="px-4 py-3 border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
          >
            <div className="flex items-start gap-2">
              <div className="flex-1 min-w-0">
                <a
                  href={item.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-gray-200 hover:text-emerald-400 transition-colors flex items-start gap-1 group"
                >
                  <span className="line-clamp-2">{item.title}</span>
                  <ExternalLink className="w-3 h-3 shrink-0 mt-0.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                </a>
                <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                  <span className="text-xs px-1.5 py-0.5 rounded bg-gray-700 text-gray-300 font-medium">
                    {item.source_name}
                  </span>
                  {item.source_category && (
                    <span className="text-xs text-gray-500">{item.source_category}</span>
                  )}
                  <span className="text-xs text-gray-600">{timeAgo(item.pub_date)}</span>
                </div>
              </div>
              <div className="flex flex-col items-end gap-1 shrink-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-gray-500">rel</span>
                  <div className="w-12 bg-gray-700 rounded-full h-1.5">
                    <div
                      className={`h-1.5 rounded-full ${relevanceColor(item.relevance_score)}`}
                      style={{ width: `${item.relevance_score * 100}%` }}
                    />
                  </div>
                </div>
                <span className={`text-xs font-mono ${truthColor(item.truth_score)}`}>
                  T:{item.truth_score.toFixed(1)}
                </span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
