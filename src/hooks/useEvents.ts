import { useState, useEffect, useCallback } from 'react'
import type { SentinelEvent, EventFilters } from '../types/sentinel'
import { fetchEvents, getConfig } from '../api/client'
import { useSSE } from './useSSE'

export function useEvents(filters: EventFilters = {}) {
  const [events, setEvents] = useState<SentinelEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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
    setEvents(prev => [event, ...prev].slice(0, 500))
  }, [])

  useSSE(handleSSE, getConfig().configured)

  return { events, loading, error, reload: load }
}
