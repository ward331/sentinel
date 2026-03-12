import { useEffect, useRef, useCallback } from 'react'
import type { SentinelEvent } from '../types/sentinel'
import { sseUrl, getConfig } from '../api/client'

type SSECallback = (event: SentinelEvent) => void

export function useSSE(onEvent: SSECallback, enabled: boolean = true) {
  const esRef = useRef<EventSource | null>(null)
  const cbRef = useRef(onEvent)
  const bufferRef = useRef<SentinelEvent[]>([])
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  cbRef.current = onEvent

  // Batch SSE events — flush at most every 2 seconds
  const flush = useCallback(() => {
    timerRef.current = null
    const buf = bufferRef.current
    if (buf.length === 0) return
    bufferRef.current = []
    // Only deliver the most recent events from the batch
    for (const event of buf.slice(-20)) {
      cbRef.current(event)
    }
  }, [])

  const connect = useCallback(() => {
    if (!getConfig().configured || !enabled) return
    if (esRef.current) esRef.current.close()

    const es = new EventSource(sseUrl())
    esRef.current = es

    const handleEvent = (e: MessageEvent) => {
      try {
        const event: SentinelEvent = JSON.parse(e.data)
        bufferRef.current.push(event)
        if (!timerRef.current) {
          timerRef.current = setTimeout(flush, 2000)
        }
      } catch { /* ignore parse errors */ }
    }

    es.addEventListener('new', handleEvent)
    es.addEventListener('message', handleEvent)

    es.onerror = () => {
      es.close()
      setTimeout(connect, 5000)
    }
  }, [enabled, flush])

  useEffect(() => {
    connect()
    return () => {
      esRef.current?.close()
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [connect])
}
