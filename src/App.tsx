import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { MapContainer, TileLayer, useMap } from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { getConfig, clearConfig, fetchEvents, sseUrl } from './api/client'
import { SetupWizard } from './components/Setup/SetupWizard'
import { Header, type View } from './components/Layout/Header'
import { FilterPanel, CATEGORY_TO_GROUP_COLOR } from './components/Filters/FilterPanel'
import { EventDetail } from './components/Layout/EventDetail'
import { ProviderHealth } from './components/Health/ProviderHealth'
import { AlertRules } from './components/Alerts/AlertRules'
import { SettingsPage } from './components/Settings/SettingsPage'
import type { SentinelEvent, EventFilters } from './types/sentinel'

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ef4444', high: '#f97316', medium: '#eab308', low: '#22c55e',
}

// Use group-based colors from FilterPanel for consistency
const CATEGORY_COLORS: Record<string, string> = CATEGORY_TO_GROUP_COLOR

// ─── Imperative marker layer ────────────────────────────────────────
function MarkerManager({ events, onSelect }: { events: SentinelEvent[], onSelect: (e: SentinelEvent) => void }) {
  const map = useMap()
  const groupRef = useRef<L.LayerGroup>(L.layerGroup())
  const onSelectRef = useRef(onSelect)
  onSelectRef.current = onSelect

  useEffect(() => {
    groupRef.current.addTo(map)
    return () => { groupRef.current.remove() }
  }, [map])

  useEffect(() => {
    const group = groupRef.current
    group.clearLayers()

    for (const event of events.slice(0, 500)) {
      const c = event.location?.coordinates
      if (!c || !Array.isArray(c) || c.length < 2) continue

      let lat: number, lon: number
      if (event.location.type === 'Polygon' && Array.isArray(c[0])) {
        const ring = c as number[][]
        let sLat = 0, sLon = 0
        for (const pt of ring) { sLon += pt[0]; sLat += pt[1] }
        lat = sLat / ring.length; lon = sLon / ring.length
      } else {
        lat = c[1] as number; lon = c[0] as number
      }

      if (typeof lat !== 'number' || typeof lon !== 'number' || isNaN(lat) || isNaN(lon)) continue

      const catColor = CATEGORY_COLORS[event.category] || SEVERITY_COLORS[event.severity] || '#666'
      const sevColor = SEVERITY_COLORS[event.severity] || '#666'
      const radius = event.severity === 'critical' ? 10 : event.severity === 'high' ? 8 : 6

      const marker = L.circleMarker([lat, lon], {
        radius, color: sevColor, fillColor: catColor, fillOpacity: 0.7, weight: 2,
      })

      marker.bindPopup(`
        <div style="min-width:180px">
          <div style="display:flex;align-items:center;gap:6px;margin-bottom:4px">
            <span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${catColor}"></span>
            <strong style="font-size:13px">${event.title}</strong>
          </div>
          <p style="font-size:11px;color:#999;margin:0">${event.source} · ${event.category}</p>
          ${event.magnitude > 0 ? `<p style="font-size:11px;font-family:monospace;margin:4px 0 0">M${event.magnitude.toFixed(1)}</p>` : ''}
          <p style="font-size:10px;color:#777;margin:4px 0 0">${new Date(event.occurred_at).toLocaleString()}</p>
        </div>
      `)

      const ev = event
      marker.on('click', () => onSelectRef.current(ev))
      group.addLayer(marker)
    }
  }, [events])

  return null
}

function InvalidateSize() {
  const map = useMap()
  useEffect(() => {
    const t = setTimeout(() => map.invalidateSize(), 200)
    return () => clearTimeout(t)
  }, [map])
  return null
}

// ─── SSE with batching + client-side filtering ──────────────────────
function useThrottledSSE(enabled: boolean, onEvent: (event: SentinelEvent) => void) {
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  useEffect(() => {
    if (!enabled || !getConfig().configured) return

    const es = new EventSource(sseUrl())
    const buffer: SentinelEvent[] = []
    let timer: ReturnType<typeof setTimeout> | null = null

    function flush() {
      timer = null
      const batch = buffer.splice(0, buffer.length)
      for (const e of batch.slice(-10)) {
        onEventRef.current(e)
      }
    }

    function handleMsg(e: MessageEvent) {
      try {
        buffer.push(JSON.parse(e.data))
        if (!timer) timer = setTimeout(flush, 3000)
      } catch {}
    }

    es.addEventListener('new', handleMsg)
    es.addEventListener('message', handleMsg)
    es.onerror = () => { es.close() }

    return () => {
      es.close()
      if (timer) clearTimeout(timer)
    }
  }, [enabled])
}

// ─── Client-side filter ─────────────────────────────────────────────
function applyClientFilters(events: SentinelEvent[], filters: EventFilters, selectedCategories: Set<string>): SentinelEvent[] {
  return events.filter(e => {
    if (selectedCategories.size > 0 && !selectedCategories.has(e.category)) return false
    if (filters.severity && e.severity !== filters.severity) return false
    if (filters.source && e.source !== filters.source) return false
    if (filters.min_magnitude && e.magnitude < filters.min_magnitude) return false
    if (filters.q) {
      const q = filters.q.toLowerCase()
      if (!e.title.toLowerCase().includes(q) && !e.description?.toLowerCase().includes(q) && !e.source.toLowerCase().includes(q)) return false
    }
    return true
  })
}

// ─── Main App ───────────────────────────────────────────────────────
function App() {
  const [configured, setConfigured] = useState(getConfig().configured)
  const [view, setView] = useState<View>('map')
  const [filters, setFilters] = useState<EventFilters>({ exclude_category: 'satellite' })
  const [allEvents, setAllEvents] = useState<SentinelEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedEvent, setSelectedEvent] = useState<SentinelEvent | null>(null)
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set())

  // Fetch from API (server-side: exclude_category only)
  useEffect(() => {
    if (!configured) return
    let cancelled = false
    setLoading(true)
    fetchEvents({ exclude_category: filters.exclude_category, limit: 500 })
      .then(res => { if (!cancelled) setAllEvents(res.events || []) })
      .catch(() => {})
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [configured, filters.exclude_category])

  // SSE
  useThrottledSSE(configured, useCallback((event: SentinelEvent) => {
    setAllEvents(prev => [event, ...prev].slice(0, 500))
  }, []))

  // All unique categories from fetched data
  const allCategories = useMemo(() =>
    Array.from(new Set(allEvents.map(e => e.category).filter(Boolean))).sort(),
    [allEvents]
  )

  const sources = useMemo(() =>
    Array.from(new Set(allEvents.map(e => e.source).filter(Boolean))).sort(),
    [allEvents]
  )

  // Event counts by category
  const eventCountsByCategory = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of allEvents) {
      counts[e.category] = (counts[e.category] || 0) + 1
    }
    return counts
  }, [allEvents])

  // Client-side filtered events
  const events = useMemo(() =>
    applyClientFilters(allEvents, filters, selectedCategories),
    [allEvents, filters, selectedCategories]
  )

  const toggleCategory = useCallback((cat: string) => {
    setSelectedCategories(prev => {
      const next = new Set(prev)
      if (next.has(cat)) next.delete(cat)
      else next.add(cat)
      return next
    })
  }, [])

  const clearCategories = useCallback(() => setSelectedCategories(new Set()), [])
  const selectAllCategories = useCallback(() => setSelectedCategories(new Set(allCategories)), [allCategories])

  if (!configured) {
    return <SetupWizard onComplete={() => setConfigured(true)} />
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header
        view={view}
        onViewChange={setView}
        onOpenSettings={() => setView(view === 'settings' ? 'map' : 'settings')}
        connected={configured}
        eventCount={events.length}
      />

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {view === 'map' && (
          <>
            {/* Sidebar */}
            <div style={{ width: 320, display: 'flex', flexDirection: 'column', flexShrink: 0 }}
                 className="border-r border-gray-800 bg-gray-900">
              <FilterPanel
                filters={filters}
                onChange={setFilters}
                sources={sources}
                categories={allCategories}
                selectedCategories={selectedCategories}
                onToggleCategory={toggleCategory}
                onClearCategories={clearCategories}
                onSelectAllCategories={selectAllCategories}
                eventCounts={eventCountsByCategory}
              />
              <div className="flex-1 overflow-y-auto event-feed">
                {loading && allEvents.length === 0 && (
                  <div className="p-4 text-center text-gray-500 text-sm">Loading events...</div>
                )}
                {!loading && events.length === 0 && (
                  <div className="p-4 text-center text-gray-500 text-sm">No events match filters</div>
                )}
                {events.slice(0, 100).map(e => (
                  <div
                    key={e.id}
                    onClick={() => setSelectedEvent(e)}
                    className={`px-3 py-2 border-b border-gray-800 cursor-pointer hover:bg-gray-800/50 transition-colors ${
                      selectedEvent?.id === e.id ? 'bg-gray-800' : ''
                    }`}
                  >
                    <div className="flex items-center gap-2 mb-0.5">
                      <span className="w-2 h-2 rounded-full shrink-0"
                            style={{ background: CATEGORY_COLORS[e.category] || SEVERITY_COLORS[e.severity] || '#666' }} />
                      <span className="text-sm font-medium text-gray-200 truncate">{e.title}</span>
                    </div>
                    <div className="flex items-center gap-1.5 text-xs text-gray-500 pl-4">
                      <span>{e.source}</span>
                      <span>·</span>
                      <span style={{ color: CATEGORY_COLORS[e.category] || '#6b7280' }}>{e.category.replace(/_/g, ' ')}</span>
                      {e.magnitude > 0 && <><span>·</span><span>M{e.magnitude.toFixed(1)}</span></>}
                      <span>·</span>
                      <span className={
                        e.severity === 'critical' ? 'text-red-400' :
                        e.severity === 'high' ? 'text-orange-400' :
                        e.severity === 'medium' ? 'text-yellow-400' : 'text-green-400'
                      }>{e.severity}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Map */}
            <div style={{ flex: 1, position: 'relative' }}>
              <div style={{ position: 'absolute', inset: 0 }}>
                <MapContainer center={[20, 0]} zoom={3} style={{ width: '100%', height: '100%' }} zoomControl={true}>
                  <TileLayer url="https://tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="OSM" maxZoom={19} />
                  <InvalidateSize />
                  <MarkerManager events={events} onSelect={setSelectedEvent} />
                </MapContainer>
              </div>
              {selectedEvent && (
                <EventDetail event={selectedEvent} onClose={() => setSelectedEvent(null)} />
              )}
            </div>
          </>
        )}

        {view === 'health' && <div className="flex-1"><ProviderHealth /></div>}
        {view === 'alerts' && <div className="flex-1"><AlertRules /></div>}
        {view === 'settings' && (
          <SettingsPage onDisconnect={() => { clearConfig(); setConfigured(false) }} />
        )}
      </div>
    </div>
  )
}

export default App
