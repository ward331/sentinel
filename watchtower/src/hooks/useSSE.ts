import { useEffect, useRef, useCallback } from 'react'
import type { SentinelEvent } from '../types/sentinel'
import { sseUrl, getConfig } from '../api/client'

type SSECallback = (event: SentinelEvent) => void

export function useSSE(onEvent: SSECallback, enabled: boolean = true) {
  const esRef = useRef<EventSource | null>(null)
  const cbRef = useRef(onEvent)
  cbRef.current = onEvent

  const connect = useCallback(() => {
    if (!getConfig().configured || !enabled) return
    if (esRef.current) esRef.current.close()

    const es = new EventSource(sseUrl())
    esRef.current = es

    es.addEventListener('new', (e) => {
      try {
        const event: SentinelEvent = JSON.parse(e.data)
        cbRef.current(event)
      } catch { /* ignore parse errors */ }
    })

    es.addEventListener('message', (e) => {
      try {
        const event: SentinelEvent = JSON.parse(e.data)
        cbRef.current(event)
      } catch { /* ignore */ }
    })

    es.onerror = () => {
      es.close()
      setTimeout(connect, 5000)
    }
  }, [enabled])

  useEffect(() => {
    connect()
    return () => { esRef.current?.close() }
  }, [connect])
}
