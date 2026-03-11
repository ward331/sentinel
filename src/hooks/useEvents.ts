import { useState, useEffect, useCallback, useRef } from 'react'
import type { SentinelEvent, EventFilters } from '../types/sentinel'
import { fetchEvents, getConfig } from '../api/client'
import { useSSE } from './useSSE'

export function useEvents(filters: EventFilters = {}) {
  const [events, setEvents] = useState<SentinelEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const filtersRef = useRef(filters)
  filtersRef.current = filters

  const load = useCallback(async () => {
    if (!getConfig().configured) return
    setLoading(true)
    setError(null)
    try {
      const res = await fetchEvents(filters)
      setEvents(res.events || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch events')
    } finally {
      setLoading(false)
    }
  }, [JSON.stringify(filters)])

  useEffect(() => { load() }, [load])

  const handleSSE = useCallback((event: SentinelEvent) => {
    const f = filtersRef.current
    // Apply client-side filters to SSE events
    if (f.exclude_category) {
      const excluded = f.exclude_category.split(',').map(s => s.trim())
      if (excluded.includes(event.category)) return
    }
    if (f.exclude_source) {
      const excluded = f.exclude_source.split(',').map(s => s.trim())
      if (excluded.includes(event.source)) return
    }
    if (f.category && event.category !== f.category) return
    if (f.source && event.source !== f.source) return
    if (f.severity && event.severity !== f.severity) return
    if (f.min_magnitude && event.magnitude < f.min_magnitude) return

    setEvents(prev => [event, ...prev].slice(0, 500))
  }, [])

  useSSE(handleSSE, getConfig().configured)

  return { events, loading, error, reload: load }
}
