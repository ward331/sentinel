import { useState, useRef, useCallback, useEffect } from 'react';
import {
  Search,
  Loader2,
  AlertCircle,
  MapPin,
} from 'lucide-react';
import { searchEntities } from '../../api/client';
import type { EntitySearchResult } from '../../types/sentinel';

const TYPE_COLORS: Record<string, string> = {
  event: 'bg-red-400/20 text-red-400 border-red-400/30',
  aircraft: 'bg-orange-400/20 text-orange-400 border-orange-400/30',
  vessel: 'bg-yellow-400/20 text-yellow-400 border-yellow-400/30',
  satellite: 'bg-emerald-400/20 text-emerald-400 border-emerald-400/30',
};

function formatCoord(val: number | null | undefined): string {
  if (val == null) return '--';
  return val.toFixed(4);
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function EntitySearch({
  onSelectLocation,
}: {
  onSelectLocation?: (lat: number, lon: number) => void;
}) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<EntitySearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const doSearch = useCallback(async (q: string) => {
    const trimmed = q.trim();
    if (!trimmed) {
      setResults([]);
      setSearched(false);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await searchEntities(trimmed);
      setResults(data.results || []);
      setSearched(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Debounced search
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      doSearch(query);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [query, doSearch]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      doSearch(query);
    }
  };

  const handleResultClick = (result: EntitySearchResult) => {
    if (onSelectLocation && result.lat != null && result.lon != null) {
      onSelectLocation(result.lat, result.lon);
    }
  };

  return (
    <div className="bg-gray-900 rounded-lg p-4 space-y-3">
      {/* Search input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Search entities and events..."
          className="w-full bg-gray-800 text-gray-100 rounded pl-9 pr-3 py-2 text-sm border border-gray-700 focus:outline-none focus:border-emerald-400 placeholder-gray-500"
        />
        {loading && (
          <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 animate-spin" />
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="flex items-center gap-2 text-sm text-red-400">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {/* Results */}
      {searched && !error && (
        <div className="space-y-1.5">
          <div className="text-xs text-gray-500">
            {results.length} result{results.length !== 1 ? 's' : ''}
          </div>

          {results.length === 0 ? (
            <div className="text-center py-4 text-gray-500 text-sm">
              No results found
            </div>
          ) : (
            <div className="max-h-80 overflow-y-auto space-y-1.5 scrollbar-thin">
              {results.map((result, idx) => {
                const hasLocation = result.lat != null && result.lon != null;
                const clickable = hasLocation && !!onSelectLocation;

                return (
                  <button
                    key={result.id ?? idx}
                    type="button"
                    onClick={() => handleResultClick(result)}
                    disabled={!clickable}
                    className={`w-full text-left flex items-start gap-3 bg-gray-800 rounded px-3 py-2.5 transition-colors ${
                      clickable
                        ? 'hover:bg-gray-750 hover:bg-gray-700 cursor-pointer'
                        : 'cursor-default'
                    }`}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm text-gray-100 truncate">{result.name}</span>
                        <span
                          className={`shrink-0 px-1.5 py-0.5 text-[10px] rounded border ${
                            TYPE_COLORS[result.type] || 'bg-gray-700 text-gray-400 border-gray-600'
                          }`}
                        >
                          {result.type}
                        </span>
                      </div>
                      <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
                        {result.source && <span>{result.source}</span>}
                        {result.source && hasLocation && <span>&middot;</span>}
                        {hasLocation && (
                          <span className="flex items-center gap-0.5">
                            <MapPin className="w-3 h-3" />
                            {formatCoord(result.lat)}, {formatCoord(result.lon)}
                          </span>
                        )}
                        {result.last_seen && (
                          <>
                            <span>&middot;</span>
                            <span>{timeAgo(result.last_seen)}</span>
                          </>
                        )}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
