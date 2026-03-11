import { useState, useRef, useCallback } from 'react'
import { Search, MapPin, X, Crosshair } from 'lucide-react'

interface FindLocateBarProps {
  onFlyTo: (coords: [number, number]) => void
}

interface SearchResult {
  display_name: string
  lat: string
  lon: string
}

function parseCoordinates(input: string): [number, number] | null {
  // Try "lat, lon" format
  const parts = input
    .replace(/[°'"NSEW]/gi, '')
    .split(/[,\s]+/)
    .filter(Boolean)

  if (parts.length === 2) {
    const a = parseFloat(parts[0])
    const b = parseFloat(parts[1])
    if (!isNaN(a) && !isNaN(b) && Math.abs(a) <= 90 && Math.abs(b) <= 180) {
      // Return as [lon, lat] for map compatibility
      return [b, a]
    }
  }
  return null
}

export default function FindLocateBar({ onFlyTo }: FindLocateBarProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const doSearch = useCallback(async (q: string) => {
    if (q.length < 2) {
      setResults([])
      return
    }

    // Check for direct coordinates first
    const coords = parseCoordinates(q)
    if (coords) {
      setResults([
        {
          display_name: `Coordinates: ${q}`,
          lon: coords[0].toString(),
          lat: coords[1].toString(),
        },
      ])
      setIsOpen(true)
      return
    }

    setLoading(true)
    try {
      const url = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(q)}&format=json&limit=5`
      const res = await fetch(url, {
        headers: { 'User-Agent': 'SentinelWatchtower/4.0' },
      })
      const data: SearchResult[] = await res.json()
      setResults(data)
      setIsOpen(data.length > 0)
    } catch {
      setResults([])
    } finally {
      setLoading(false)
    }
  }, [])

  const handleInput = (value: string) => {
    setQuery(value)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => doSearch(value), 400)
  }

  const handleSelect = (result: SearchResult) => {
    const lon = parseFloat(result.lon)
    const lat = parseFloat(result.lat)
    if (!isNaN(lon) && !isNaN(lat)) {
      onFlyTo([lon, lat])
    }
    setQuery('')
    setResults([])
    setIsOpen(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && results.length > 0) {
      handleSelect(results[0])
    }
    if (e.key === 'Escape') {
      setIsOpen(false)
      setQuery('')
    }
  }

  return (
    <div className="absolute top-3 left-1/2 -translate-x-1/2 z-30 w-[400px]">
      <div className="relative">
        <div className="flex items-center bg-gray-950/90 border border-gray-800 rounded-lg backdrop-blur-sm overflow-hidden focus-within:border-gray-700 transition-colors">
          <div className="pl-3 pr-2">
            {loading ? (
              <div className="w-3 h-3 border border-cyan-600 border-t-transparent rounded-full animate-spin" />
            ) : (
              <Search size={14} className="text-gray-600" />
            )}
          </div>
          <input
            type="text"
            value={query}
            onChange={(e) => handleInput(e.target.value)}
            onKeyDown={handleKeyDown}
            onFocus={() => results.length > 0 && setIsOpen(true)}
            placeholder="Enter coordinates or place name..."
            className="flex-1 bg-transparent text-[11px] font-mono text-gray-300 placeholder-gray-700 py-2 pr-2 outline-none"
          />
          {query && (
            <button
              onClick={() => {
                setQuery('')
                setResults([])
                setIsOpen(false)
              }}
              className="pr-3 text-gray-600 hover:text-gray-400 transition-colors"
            >
              <X size={12} />
            </button>
          )}
        </div>

        {/* Dropdown results */}
        {isOpen && results.length > 0 && (
          <div className="absolute top-full left-0 right-0 mt-1 bg-gray-950/95 border border-gray-800 rounded-lg backdrop-blur-sm overflow-hidden shadow-xl">
            {results.map((r, i) => (
              <button
                key={i}
                onClick={() => handleSelect(r)}
                className="w-full flex items-start gap-2 px-3 py-2 hover:bg-gray-900/80 transition-colors text-left border-b border-gray-900 last:border-0"
              >
                {r.display_name.startsWith('Coordinates') ? (
                  <Crosshair size={12} className="text-cyan-500 mt-0.5 shrink-0" />
                ) : (
                  <MapPin size={12} className="text-gray-600 mt-0.5 shrink-0" />
                )}
                <div className="flex-1 min-w-0">
                  <span className="text-[11px] text-gray-300 line-clamp-2">
                    {r.display_name}
                  </span>
                  <span className="text-[9px] font-mono text-gray-600 block mt-0.5">
                    {parseFloat(r.lat).toFixed(4)}, {parseFloat(r.lon).toFixed(4)}
                  </span>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
