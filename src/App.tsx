import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { MapContainer, TileLayer, useMap } from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { getConfig, clearConfig, fetchEvents, fetchCorrelations, sseUrl } from './api/client'
import { SetupWizard } from './components/Setup/SetupWizard'
import { Header, type View } from './components/Layout/Header'
import { FilterPanel, CATEGORY_TO_GROUP_COLOR } from './components/Filters/FilterPanel'
import { EventDetail } from './components/Layout/EventDetail'
import { ProviderHealth } from './components/Health/ProviderHealth'
import { AlertRules } from './components/Alerts/AlertRules'
import { SettingsPage } from './components/Settings/SettingsPage'
import { SignalBoard } from './components/Intel/SignalBoard'
import { IntelBriefing } from './components/Intel/IntelBriefing'
import { NewsFeed } from './components/Intel/NewsFeed'
import { CorrelationList } from './components/Intel/CorrelationList'
import { FinancialDashboard } from './components/Financial/FinancialDashboard'
import { OsintBrowser } from './components/OSINT/OsintBrowser'
import { NotificationSettings } from './components/Notifications/NotificationSettings'
import { ProximityPanel } from './components/Proximity/ProximityPanel'
import { EntitySearch } from './components/Search/EntitySearch'
import type { SentinelEvent, EventFilters, CorrelationFlash } from './types/sentinel'

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ef4444', high: '#f97316', medium: '#eab308', low: '#22c55e',
}

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
          ${event.truth_score ? `<p style="font-size:10px;color:#6ee;margin:4px 0 0">Truth: ${event.truth_score}%</p>` : ''}
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

// ─── Correlation overlay on map ──────────────────────────────────────
function CorrelationOverlay({ correlations }: { correlations: CorrelationFlash[] }) {
  const map = useMap()
  const groupRef = useRef<L.LayerGroup>(L.layerGroup())

  useEffect(() => {
    groupRef.current.addTo(map)
    return () => { groupRef.current.remove() }
  }, [map])

  useEffect(() => {
    const group = groupRef.current
    group.clearLayers()

    for (const c of correlations) {
      if (!c.lat || !c.lon) continue
      const intensity = Math.min(c.event_count / 10, 1)
      const color = c.confirmed ? '#ef4444' : '#f97316'
      const circle = L.circle([c.lat, c.lon], {
        radius: (c.radius_km || 50) * 1000,
        color,
        fillColor: color,
        fillOpacity: 0.08 + intensity * 0.12,
        weight: 2,
        dashArray: c.confirmed ? undefined : '8 4',
      })
      circle.bindPopup(`
        <div style="min-width:160px">
          <strong style="font-size:13px;color:${color}">${c.region_name || 'Correlation'}</strong>
          <p style="font-size:11px;color:#999;margin:4px 0 0">${c.event_count} events from ${c.source_count} sources</p>
          <p style="font-size:10px;color:#777;margin:2px 0 0">Radius: ${c.radius_km?.toFixed(0) || '?'} km</p>
          ${c.confirmed ? '<p style="font-size:10px;color:#ef4444;margin:2px 0 0">⚡ CONFIRMED</p>' : ''}
        </div>
      `)
      group.addLayer(circle)
    }
  }, [correlations])

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

// ─── Fly to location helper ─────────────────────────────────────────
function FlyTo({ lat, lon }: { lat: number; lon: number }) {
  const map = useMap()
  useEffect(() => {
    map.flyTo([lat, lon], 8, { duration: 1.5 })
  }, [map, lat, lon])
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
  const [correlations, setCorrelations] = useState<CorrelationFlash[]>([])
  const [flyTarget, setFlyTarget] = useState<{ lat: number; lon: number } | null>(null)
  const [showSearch, setShowSearch] = useState(false)

  // Fetch events
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

  // Fetch correlations for map overlay
  useEffect(() => {
    if (!configured) return
    let cancelled = false
    function load() {
      fetchCorrelations()
        .then(res => { if (!cancelled) setCorrelations(res.correlations || []) })
        .catch(() => {})
    }
    load()
    const iv = setInterval(load, 30000)
    return () => { cancelled = true; clearInterval(iv) }
  }, [configured])

  // SSE
  useThrottledSSE(configured, useCallback((event: SentinelEvent) => {
    setAllEvents(prev => [event, ...prev].slice(0, 500))
  }, []))

  const allCategories = useMemo(() =>
    Array.from(new Set(allEvents.map(e => e.category).filter(Boolean))).sort(),
    [allEvents]
  )

  const sources = useMemo(() =>
    Array.from(new Set(allEvents.map(e => e.source).filter(Boolean))).sort(),
    [allEvents]
  )

  const eventCountsByCategory = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of allEvents) {
      counts[e.category] = (counts[e.category] || 0) + 1
    }
    return counts
  }, [allEvents])

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

  const handleCorrelationSelect = useCallback((c: CorrelationFlash) => {
    if (c.lat && c.lon) {
      setFlyTarget({ lat: c.lat, lon: c.lon })
      setView('map')
    }
  }, [])

  const handleSearchLocation = useCallback((lat: number, lon: number) => {
    setFlyTarget({ lat, lon })
    setShowSearch(false)
    setView('map')
  }, [])

  if (!configured) {
    return <SetupWizard onComplete={() => setConfigured(true)} />
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Header
        view={view}
        onViewChange={(v) => { setView(v); setShowSearch(false) }}
        onOpenSettings={() => { setView(view === 'settings' ? 'map' : 'settings'); setShowSearch(false) }}
        connected={configured}
        eventCount={events.length}
      />

      {/* Global search bar */}
      <div className="bg-gray-900 border-b border-gray-800 px-4 py-1 flex items-center gap-2">
        <button
          onClick={() => setShowSearch(!showSearch)}
          className="text-xs text-gray-500 hover:text-gray-300 flex items-center gap-1 px-2 py-1 rounded hover:bg-gray-800 transition-colors"
        >
          🔍 Search entities...
        </button>
        {showSearch && (
          <div className="flex-1 max-w-md">
            <EntitySearch onSelectLocation={handleSearchLocation} />
          </div>
        )}
      </div>

      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        {/* ── Map View ── */}
        {view === 'map' && (
          <>
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
                      {e.truth_score > 0 && <><span>·</span><span className="text-cyan-400">T{e.truth_score}</span></>}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div style={{ flex: 1, position: 'relative' }}>
              <div style={{ position: 'absolute', inset: 0 }}>
                <MapContainer center={[20, 0]} zoom={3} style={{ width: '100%', height: '100%' }} zoomControl={true}>
                  <TileLayer url="https://tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="OSM" maxZoom={19} />
                  <InvalidateSize />
                  <MarkerManager events={events} onSelect={setSelectedEvent} />
                  <CorrelationOverlay correlations={correlations} />
                  {flyTarget && <FlyTo lat={flyTarget.lat} lon={flyTarget.lon} />}
                </MapContainer>
              </div>

              {/* Correlation sidebar on map */}
              {correlations.length > 0 && (
                <div className="absolute top-2 right-2 z-[1000] w-64 max-h-64 overflow-y-auto bg-gray-900/95 border border-gray-700 rounded-lg shadow-lg backdrop-blur-sm">
                  <div className="px-3 py-2 border-b border-gray-700 text-xs font-semibold text-orange-400 flex items-center gap-1.5">
                    ⚡ {correlations.length} Active Correlation{correlations.length !== 1 ? 's' : ''}
                  </div>
                  {correlations.slice(0, 5).map(c => (
                    <button
                      key={c.id}
                      onClick={() => handleCorrelationSelect(c)}
                      className="w-full text-left px-3 py-2 border-b border-gray-800 hover:bg-gray-800/50 transition-colors"
                    >
                      <div className="text-xs font-medium text-gray-200 truncate">{c.region_name || 'Unknown'}</div>
                      <div className="text-xs text-gray-500">{c.event_count} events · {c.source_count} sources · {c.radius_km?.toFixed(0)}km</div>
                    </button>
                  ))}
                </div>
              )}

              {selectedEvent && (
                <EventDetail event={selectedEvent} onClose={() => setSelectedEvent(null)} />
              )}
            </div>
          </>
        )}

        {/* ── Intel View ── */}
        {view === 'intel' && (
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <SignalBoard />
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="space-y-4">
                <IntelBriefing />
              </div>
              <div className="space-y-4">
                <NewsFeed />
                <CorrelationList onSelect={handleCorrelationSelect} />
              </div>
            </div>
          </div>
        )}

        {/* ── Financial View ── */}
        {view === 'financial' && (
          <div className="flex-1 overflow-y-auto">
            <FinancialDashboard />
          </div>
        )}

        {/* ── Health View ── */}
        {view === 'health' && <div className="flex-1 overflow-y-auto"><ProviderHealth /></div>}

        {/* ── Alerts View ── */}
        {view === 'alerts' && <div className="flex-1 overflow-y-auto"><AlertRules /></div>}

        {/* ── OSINT View ── */}
        {view === 'osint' && <div className="flex-1 overflow-y-auto"><OsintBrowser /></div>}

        {/* ── Settings View ── */}
        {view === 'settings' && (
          <div className="flex-1 overflow-y-auto">
            <SettingsPage onDisconnect={() => { clearConfig(); setConfigured(false) }} />
            <div className="border-t border-gray-800 mt-4">
              <div className="p-4">
                <h2 className="text-lg font-semibold text-gray-200 mb-4">Notification Channels</h2>
                <NotificationSettings />
              </div>
            </div>
            <div className="border-t border-gray-800">
              <div className="p-4">
                <h2 className="text-lg font-semibold text-gray-200 mb-4">Proximity Alerts</h2>
                <ProximityPanel />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default App
