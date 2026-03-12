import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { getConfig, clearConfig, fetchEvents, fetchSignalBoard, fetchCorrelations, fetchHealth, sseUrl } from './api/client'
import { SetupWizard } from './components/Setup/SetupWizard'
import { Header, type View } from './components/Layout/Header'
import { EventDetail } from './components/Layout/EventDetail'
import WorldviewLeftPanel, { type MapStyleKey } from './components/Panels/WorldviewLeftPanel'
import WorldviewRightPanel from './components/Panels/WorldviewRightPanel'
import FindLocateBar from './components/Panels/FindLocateBar'
import StatusBar from './components/Panels/StatusBar'
import MarketsPanel from './components/Panels/MarketsPanel'
import MaplibreViewer from './components/Map/MaplibreViewer'
import { ProviderHealth } from './components/Health/ProviderHealth'
import { AlertRules } from './components/Alerts/AlertRules'
import { SettingsPage } from './components/Settings/SettingsPage'
import { SignalBoard as SignalBoardView } from './components/Intel/SignalBoard'
import { IntelBriefing } from './components/Intel/IntelBriefing'
import { NewsFeed } from './components/Intel/NewsFeed'
import { CorrelationList } from './components/Intel/CorrelationList'
import { FinancialDashboard } from './components/Financial/FinancialDashboard'
import { OsintBrowser } from './components/OSINT/OsintBrowser'
import type { SentinelEvent, EventFilters, SignalBoard, CorrelationFlash, HealthResponse } from './types/sentinel'
import type { LiveData } from './types/livedata'

// Proxied through Vite: /osint/* → http://127.0.0.1:8000/api/*
const DATA_FETCHER_BASE = '/osint'

// ─── SSE with batching ─────────────────────────────────────────────────
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
      } catch { /* ignore parse errors */ }
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

// ─── Region name lookup for correlations ─────────────────────────────
const REGION_COORDS: [string, number, number][] = [
  ['Middle East', 32, 44], ['Ukraine', 48.4, 31.2], ['Russia', 61.5, 105.3],
  ['China', 35.9, 104.2], ['Iran', 32.4, 53.7], ['Israel/Palestine', 31.5, 34.9],
  ['Syria', 35, 38.5], ['Iraq', 33.2, 43.7], ['Afghanistan', 33.9, 67.7],
  ['Pakistan', 30.4, 69.3], ['India', 20.6, 79], ['Japan', 36.2, 138.3],
  ['North Korea', 40.3, 127.5], ['South Korea', 35.9, 127.8], ['Taiwan', 23.7, 121],
  ['Philippines', 12.9, 121.8], ['Turkey', 39.9, 32.9], ['Saudi Arabia', 23.9, 45.1],
  ['Yemen', 15.6, 48.5], ['Somalia', 5.2, 46.2], ['Nigeria', 9.1, 8.7],
  ['Ethiopia', 9.1, 40.5], ['Sudan', 12.9, 30.2], ['Libya', 26.3, 17.2],
  ['Egypt', 26.8, 30.8], ['United States', 39.8, -98.6], ['Mexico', 23.6, -102.6],
  ['Brazil', -14.2, -51.9], ['Germany', 51.2, 10.5], ['France', 46.2, 2.2],
  ['UK', 55.4, -3.4], ['Poland', 51.9, 19.1], ['South Africa', -30.6, 22.9],
  ['Indonesia', -0.8, 113.9], ['Australia', -25.3, 133.8], ['Canada', 56.1, -106.3],
]

function _findRegionName(lat: number, lon: number): string {
  let best = 'Unknown Region'
  let bestDist = Infinity
  for (const [name, rlat, rlon] of REGION_COORDS) {
    const d = Math.sqrt((lat - rlat) ** 2 + (lon - rlon) ** 2)
    if (d < bestDist) { bestDist = d; best = name }
  }
  return best
}

// ─── Main App ───────────────────────────────────────────────────────────
function App() {
  // Data sources
  const [events, setEvents] = useState<SentinelEvent[]>([])
  const [liveData, setLiveData] = useState<LiveData | null>(null)
  const [signalBoard, setSignalBoard] = useState<SignalBoard | null>(null)
  const [correlations, setCorrelations] = useState<CorrelationFlash[]>([])
  const [health, setHealth] = useState<HealthResponse | null>(null)

  // UI state
  const [selectedEvent, setSelectedEvent] = useState<SentinelEvent | null>(null)
  const [mapStyle, setMapStyle] = useState<MapStyleKey>('default')
  const [visibleLayers, setVisibleLayers] = useState<Set<string>>(new Set([
    'events', 'aircraft', 'ships', 'satellites', 'earthquakes', 'fires', 'conflicts', 'sigint',
  ]))
  const [flyTo, setFlyTo] = useState<[number, number] | null>(null)
  const [mouseCoords, setMouseCoords] = useState<[number, number] | null>(null)
  const [currentView, setCurrentView] = useState<View>('map')
  const [isConnected, setIsConnected] = useState(false)
  const [loading, setLoading] = useState(true)

  // Setup
  const [configured, setConfigured] = useState(getConfig().configured)

  // Filters
  const [filters] = useState<EventFilters>({ limit: 500 })

  // ─── Poll SENTINEL Go backend (events every 10s) ─────────────────
  useEffect(() => {
    if (!configured) return
    let cancelled = false

    const load = async () => {
      try {
        const data = await fetchEvents(filters)
        if (!cancelled) {
          setEvents(data.events || [])
          setIsConnected(true)
          setLoading(false)
        }
      } catch {
        if (!cancelled) {
          // Go backend is down — mark connected based on Python data availability
          setIsConnected(!!liveData)
          setLoading(false)
        }
      }
    }

    load()
    const timer = setInterval(load, 10000)
    return () => { cancelled = true; clearInterval(timer) }
  }, [configured, filters])

  // ─── Generate signal board from live data (no Go backend needed) ──
  useEffect(() => {
    if (!liveData) return

    // Compute threat levels from actual data
    const militaryFlights = liveData.military_flights?.length || 0
    const gdeltConflicts = liveData.gdelt?.length || 0
    const earthquakes = liveData.earthquakes || []
    const bigQuakes = earthquakes.filter(e => e.mag >= 5).length
    const fires = liveData.firms_fires?.length || 0

    // Military: based on military flights + GDELT conflict articles
    const milScore = Math.min(5, Math.floor(
      (militaryFlights > 50 ? 2 : militaryFlights > 20 ? 1 : 0) +
      (gdeltConflicts > 100 ? 3 : gdeltConflicts > 50 ? 2 : gdeltConflicts > 10 ? 1 : 0)
    ))
    // Cyber: moderate baseline (no direct data source)
    const cyberScore = 1
    // Financial: from Fear & Greed index
    const fgi = liveData.financial?.fear_greed_index
    const finScore = fgi !== undefined && fgi !== null
      ? (fgi < 20 ? 4 : fgi < 35 ? 3 : fgi < 50 ? 2 : fgi < 65 ? 1 : 0)
      : 1
    // Natural: earthquakes + fires
    const natScore = Math.min(5, Math.floor(
      (bigQuakes >= 3 ? 3 : bigQuakes >= 1 ? 2 : 0) +
      (fires > 3000 ? 2 : fires > 1000 ? 1 : 0)
    ))
    // Health: baseline nominal
    const healthScore = 0

    setSignalBoard({
      military: milScore,
      cyber: cyberScore,
      financial: finScore,
      natural: natScore,
      health: healthScore,
      calculated_at: new Date().toISOString(),
      active_alerts: gdeltConflicts,
      active_correlations: 0,
    })
  }, [liveData])

  // ─── Generate correlations from GDELT + news geo-data ─────────────
  useEffect(() => {
    if (!liveData) return

    // Cluster GDELT events by proximity to generate correlations
    const geoEvents = [
      ...(liveData.gdelt || []).filter(g => g.lat !== 0 && g.lon !== 0),
    ]
    const newsWithGeo = (liveData.news || []).filter(n => n.lat && n.lon)

    // Simple region-based clustering
    const regions: Record<string, { lat: number; lon: number; events: number; sources: Set<string>; name: string }> = {}

    for (const evt of geoEvents) {
      // Round to 5-degree grid for clustering
      const gridKey = `${Math.round(evt.lat / 5) * 5},${Math.round(evt.lon / 5) * 5}`
      if (!regions[gridKey]) {
        // Find nearest country name
        const name = _findRegionName(evt.lat, evt.lon)
        regions[gridKey] = { lat: evt.lat, lon: evt.lon, events: 0, sources: new Set(), name }
      }
      regions[gridKey].events++
      if (evt.domain) regions[gridKey].sources.add(evt.domain)
    }

    for (const n of newsWithGeo) {
      const gridKey = `${Math.round(n.lat! / 5) * 5},${Math.round(n.lon! / 5) * 5}`
      if (!regions[gridKey]) {
        regions[gridKey] = { lat: n.lat!, lon: n.lon!, events: 0, sources: new Set(), name: _findRegionName(n.lat!, n.lon!) }
      }
      regions[gridKey].events++
      if (n.source) regions[gridKey].sources.add(n.source)
    }

    // Convert to correlations (only regions with 3+ events)
    const corrs: CorrelationFlash[] = Object.entries(regions)
      .filter(([, r]) => r.events >= 3)
      .sort((a, b) => b[1].events - a[1].events)
      .slice(0, 20)
      .map(([key, r], i) => ({
        id: i + 1,
        region_name: r.name,
        lat: r.lat,
        lon: r.lon,
        radius_km: 250,
        event_count: r.events,
        source_count: r.sources.size,
        started_at: new Date(Date.now() - 3600000).toISOString(),
        last_event_at: new Date().toISOString(),
        confirmed: r.events >= 10,
        incident_name: r.events >= 10 ? `${r.name} Activity Cluster` : '',
      }))

    setCorrelations(corrs)
  }, [liveData])

  // ─── Poll health (30s) ────────────────────────────────────────────
  useEffect(() => {
    if (!configured) return
    let cancelled = false

    const load = async () => {
      try {
        const data = await fetchHealth()
        if (!cancelled) setHealth(data)
      } catch {
        // Try Python backend health instead
        try {
          const res = await fetch(`${DATA_FETCHER_BASE}/health`)
          const pyHealth = await res.json()
          if (!cancelled) {
            setHealth({
              status: pyHealth.status === 'operational' ? 'healthy' : 'degraded',
              version: 'v4.0.1',
              uptime: pyHealth.uptime || '',
            } as HealthResponse)
            setIsConnected(true)
          }
        } catch {
          if (!cancelled) setIsConnected(false)
        }
      }
    }

    load()
    const timer = setInterval(load, 30000)
    return () => { cancelled = true; clearInterval(timer) }
  }, [configured])

  // ─── Poll Python data fetcher (fast 60s, slow 120s) ──────────────
  useEffect(() => {
    const loadFast = async () => {
      try {
        const res = await fetch(`${DATA_FETCHER_BASE}/live-data/fast`)
        const data = await res.json()
        setLiveData(prev => ({ ...prev, ...data } as LiveData))
      } catch (e) { console.warn('[V4] Data fetcher fast poll failed:', e) }
    }
    const loadSlow = async () => {
      try {
        const res = await fetch(`${DATA_FETCHER_BASE}/live-data/slow`)
        const data = await res.json()
        // Unwrap single-element arrays for space_weather and financial
        if (Array.isArray(data.space_weather)) data.space_weather = data.space_weather[0] || null
        if (Array.isArray(data.financial)) data.financial = data.financial[0] || null
        setLiveData(prev => ({ ...prev, ...data } as LiveData))
      } catch (e) { console.warn('[V4] Data fetcher slow poll failed:', e) }
    }

    loadFast()
    loadSlow()
    const fastTimer = setInterval(loadFast, 60000)
    const slowTimer = setInterval(loadSlow, 120000)
    return () => { clearInterval(fastTimer); clearInterval(slowTimer) }
  }, [])

  // ─── SSE for real-time events ─────────────────────────────────────
  useThrottledSSE(configured, useCallback((event: SentinelEvent) => {
    setEvents(prev => [event, ...prev].slice(0, 500))
  }, []))

  // ─── FlyTo: clear after 100ms ─────────────────────────────────────
  useEffect(() => {
    if (!flyTo) return
    const t = setTimeout(() => setFlyTo(null), 100)
    return () => clearTimeout(t)
  }, [flyTo])

  // ─── Computed values ──────────────────────────────────────────────
  const sourceCounts = useMemo(() => ({
    events: events.length,
    aircraft: (liveData?.commercial_flights?.length || 0) + (liveData?.military_flights?.length || 0),
    ships: liveData?.ships?.length || 0,
    satellites: liveData?.satellites?.length || 0,
    earthquakes: liveData?.earthquakes?.length || 0,
    fires: liveData?.firms_fires?.length || 0,
    conflicts: liveData?.gdelt?.length || 0,
    sigint: liveData?.kiwisdr?.length || 0,
  }), [events, liveData])

  const freshness = liveData?.freshness || {}

  // ─── Handlers ─────────────────────────────────────────────────────
  const handleToggleLayer = useCallback((layer: string) => {
    setVisibleLayers(prev => {
      const next = new Set(prev)
      if (next.has(layer)) next.delete(layer)
      else next.add(layer)
      return next
    })
  }, [])

  const handleSelectEvent = useCallback((event: SentinelEvent) => {
    setSelectedEvent(event)
    const c = event.location?.coordinates
    if (c && Array.isArray(c) && c.length >= 2) {
      if (event.location.type === 'Polygon' && Array.isArray(c[0])) {
        const ring = c as number[][]
        let sLat = 0, sLon = 0
        for (const pt of ring) { sLon += pt[0]; sLat += pt[1] }
        setFlyTo([sLon / ring.length, sLat / ring.length])
      } else {
        setFlyTo([c[0] as number, c[1] as number])
      }
    }
  }, [])

  // ─── Setup screen ─────────────────────────────────────────────────
  if (!configured) {
    return <SetupWizard onComplete={() => setConfigured(true)} />
  }

  return (
    <div className="flex flex-col w-full h-full">
      {/* Header */}
      <Header
        currentView={currentView}
        onViewChange={setCurrentView}
        isConnected={isConnected}
      />

      {/* Main content area */}
      <div className="flex-1 flex overflow-hidden relative">
        {currentView === 'map' ? (
          <div className="flex-1 relative overflow-hidden">
            {/* Full-screen map area */}
            <div className="absolute inset-0 z-0">
              <MaplibreViewer
                events={events}
                liveData={liveData}
                selectedEvent={selectedEvent}
                onSelectEvent={handleSelectEvent}
                flyTo={flyTo}
                mapStyle={mapStyle}
                visibleLayers={visibleLayers}
                correlations={correlations}
                onMouseMove={setMouseCoords}
              />
            </div>

            {/* Left panel (absolute overlay with own collapse toggle) */}
            <WorldviewLeftPanel
              visibleLayers={visibleLayers}
              onToggleLayer={handleToggleLayer}
              mapStyle={mapStyle}
              onSetMapStyle={setMapStyle}
              freshness={freshness}
              sourceCounts={sourceCounts}
              isConnected={isConnected}
            />

            {/* Find/Locate bar */}
            <FindLocateBar onFlyTo={setFlyTo} />

            {/* Right panel (WorldviewRightPanel with tabs) */}
            <WorldviewRightPanel
              events={events}
              onSelectEvent={handleSelectEvent}
              signalBoard={signalBoard}
              correlations={correlations}
              news={liveData?.news || []}
              kiwisdr={liveData?.kiwisdr || []}
              onFlyTo={setFlyTo}
            />

            {/* Markets floating widget */}
            <MarketsPanel financial={liveData?.financial || null} />

            {/* Correlation flashes overlay */}
            {correlations.length > 0 && (
              <div className="absolute top-2 left-[340px] z-10 w-56 max-h-48 overflow-y-auto bg-gray-950/90 border border-gray-800 rounded-lg backdrop-blur-sm">
                <div className="px-2.5 py-1.5 border-b border-gray-800 text-[10px] font-mono uppercase tracking-wider text-orange-400">
                  {correlations.length} CORRELATION{correlations.length !== 1 ? 'S' : ''}
                </div>
                {correlations.slice(0, 8).map(c => (
                  <button
                    key={c.id}
                    onClick={() => {
                      if (c.lat && c.lon) setFlyTo([c.lon, c.lat])
                    }}
                    className="w-full text-left px-2.5 py-1.5 border-b border-gray-800/50 hover:bg-gray-900 transition-colors"
                  >
                    <div className="text-[10px] font-mono text-gray-300 truncate">{c.region_name || 'Unknown'}</div>
                    <div className="text-[9px] font-mono text-gray-600">{c.event_count} events / {c.source_count} src / {c.radius_km?.toFixed(0)}km</div>
                  </button>
                ))}
              </div>
            )}

            {/* Signal board mini overlay */}
            {signalBoard && (
              <div className="absolute bottom-2 left-2 z-10 flex gap-1">
                {(['military', 'cyber', 'financial', 'natural', 'health'] as const).map(key => {
                  const val = signalBoard[key] ?? 0
                  const color = val >= 4 ? 'bg-red-500' : val >= 3 ? 'bg-orange-500' : val >= 2 ? 'bg-yellow-500' : 'bg-emerald-500'
                  return (
                    <div key={key} className="flex flex-col items-center gap-0.5 px-1.5 py-1 bg-gray-950/80 rounded border border-gray-800">
                      <div className={`w-2 h-2 rounded-full ${color}`} />
                      <span className="text-[8px] font-mono uppercase text-gray-500">{key.slice(0, 3)}</span>
                    </div>
                  )
                })}
              </div>
            )}

            {/* Event detail panel */}
            {selectedEvent && (
              <EventDetail event={selectedEvent} onClose={() => setSelectedEvent(null)} />
            )}

          </div>
        ) : currentView === 'intel' ? (
          <div className="flex-1 overflow-y-auto">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 p-4">
              <SignalBoardView initialData={signalBoard} />
              <IntelBriefing events={events} news={liveData?.news || []} />
              <NewsFeed initialItems={liveData?.news || []} />
              <CorrelationList initialData={correlations} />
            </div>
          </div>
        ) : currentView === 'financial' ? (
          <div className="flex-1 overflow-y-auto">
            <FinancialDashboard data={liveData?.financial || null} />
          </div>
        ) : currentView === 'health' ? (
          <div className="flex-1 overflow-y-auto">
            <ProviderHealth />
          </div>
        ) : currentView === 'alerts' ? (
          <div className="flex-1 overflow-y-auto">
            <AlertRules />
          </div>
        ) : currentView === 'osint' ? (
          <div className="flex-1 overflow-y-auto">
            <OsintBrowser />
          </div>
        ) : currentView === 'settings' ? (
          <div className="flex-1 overflow-y-auto">
            <SettingsPage onDisconnect={() => { clearConfig(); setConfigured(false) }} />
          </div>
        ) : null}
      </div>

      {/* Status bar */}
      <StatusBar
        health={
          !health || !isConnected ? 'offline' :
          health.status === 'healthy' ? 'operational' :
          health.status === 'degraded' ? 'degraded' : 'offline'
        }
        mouseCoords={mouseCoords}
        sourceCounts={sourceCounts}
      />
    </div>
  )
}

export default App
