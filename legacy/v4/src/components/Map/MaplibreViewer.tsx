import { useRef, useEffect, useCallback, useMemo } from 'react'
import maplibregl from 'maplibre-gl'
// CSS imported in index.css (must load before Tailwind)
import type { SentinelEvent, CorrelationFlash } from '../../types/sentinel'
import type { LiveData, Aircraft, Vessel, Satellite, Earthquake, Fire, GdeltEvent, KiwiSDR } from '../../types/livedata'

// ─── Types ────────────────────────────────────────────────────────

export type MapStyleKey = 'default' | 'satellite' | 'flir' | 'nvg' | 'crt'

export interface MaplibreViewerProps {
  events: SentinelEvent[]
  liveData: LiveData | null
  selectedEvent: SentinelEvent | null
  onSelectEvent: (e: SentinelEvent | null) => void
  flyTo: [number, number] | null
  mapStyle: MapStyleKey
  visibleLayers: Set<string>
  correlations: CorrelationFlash[]
  onMouseMove?: (coords: [number, number] | null) => void
}

// ─── Constants ────────────────────────────────────────────────────

const MAPTILER_KEY = import.meta.env.VITE_MAPTILER_KEY || ''

const MAP_STYLES: Record<MapStyleKey, { url: string; label: string }> = {
  default: { url: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json', label: 'DEFAULT' },
  satellite: { url: `https://api.maptiler.com/maps/hybrid/style.json?key=${MAPTILER_KEY}`, label: 'SATELLITE' },
  flir: { url: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json', label: 'FLIR' },
  nvg: { url: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json', label: 'NVG' },
  crt: { url: 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json', label: 'CRT' },
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
}

const AIRCRAFT_COLORS: Record<string, string> = {
  commercial: '#06b6d4',
  military: '#ef4444',
  private: '#f59e0b',
}

const SHIP_COLORS: Record<string, string> = {
  cargo: '#3b82f6',
  tanker: '#f97316',
  military: '#ef4444',
  passenger: '#ffffff',
  fishing: '#22c55e',
  pleasure: '#a78bfa',
  unknown: '#6b7280',
}

const SAT_COLORS: Record<string, string> = {
  military_recon: '#ef4444',
  sar: '#06b6d4',
  sigint: '#ffffff',
  navigation: '#3b82f6',
  early_warning: '#d946ef',
  commercial: '#22c55e',
  iss: '#fbbf24',
}

const DEBOUNCE_MS = 300
const VIEWPORT_BUFFER = 0.2 // 20% buffer beyond viewport

// ─── Layer IDs ────────────────────────────────────────────────────

const LAYER_IDS = {
  events: 'sentinel-events',
  eventsGlow: 'sentinel-events-glow',
  aircraft: 'aircraft-layer',
  ships: 'ships-layer',
  satellites: 'satellites-layer',
  earthquakes: 'earthquakes-layer',
  earthquakesPulse: 'earthquakes-pulse-layer',
  fires: 'fires-layer',
  conflicts: 'conflicts-layer',
  sigint: 'sigint-layer',
  correlations: 'correlations-layer',
  terminator: 'terminator-layer',
} as const

const SOURCE_IDS = {
  events: 'sentinel-events-src',
  aircraft: 'aircraft-src',
  ships: 'ships-src',
  satellites: 'satellites-src',
  earthquakes: 'earthquakes-src',
  fires: 'fires-src',
  conflicts: 'conflicts-src',
  sigint: 'sigint-src',
  correlations: 'correlations-src',
  terminator: 'terminator-src',
} as const

// Maps visibleLayers keys to their layer IDs (some have multiple layers)
const LAYER_KEY_TO_IDS: Record<string, string[]> = {
  events: [LAYER_IDS.events, LAYER_IDS.eventsGlow],
  aircraft: [LAYER_IDS.aircraft],
  ships: [LAYER_IDS.ships],
  satellites: [LAYER_IDS.satellites],
  earthquakes: [LAYER_IDS.earthquakes, LAYER_IDS.earthquakesPulse],
  fires: [LAYER_IDS.fires],
  conflicts: [LAYER_IDS.conflicts],
  sigint: [LAYER_IDS.sigint],
  correlations: [LAYER_IDS.correlations],
  terminator: [LAYER_IDS.terminator],
}

// ─── Helpers ──────────────────────────────────────────────────────

function emptyFC(): GeoJSON.FeatureCollection {
  return { type: 'FeatureCollection', features: [] }
}

/** Debounce a function call */
function debounce<T extends (...args: unknown[]) => void>(fn: T, ms: number): T & { cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | null = null
  const debounced = (...args: unknown[]) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), ms)
  }
  debounced.cancel = () => { if (timer) clearTimeout(timer) }
  return debounced as T & { cancel: () => void }
}

/** Compute viewport bounds with buffer for culling */
function getBufferedBounds(map: maplibregl.Map): maplibregl.LngLatBounds | null {
  const bounds = map.getBounds()
  if (!bounds) return null
  const sw = bounds.getSouthWest()
  const ne = bounds.getNorthEast()
  const lngSpan = ne.lng - sw.lng
  const latSpan = ne.lat - sw.lat
  const buf = VIEWPORT_BUFFER
  return new maplibregl.LngLatBounds(
    [sw.lng - lngSpan * buf, Math.max(-90, sw.lat - latSpan * buf)],
    [ne.lng + lngSpan * buf, Math.min(90, ne.lat + latSpan * buf)]
  )
}

/** Check if a point is within bounds */
function inBounds(lon: number, lat: number, bounds: maplibregl.LngLatBounds): boolean {
  const sw = bounds.getSouthWest()
  const ne = bounds.getNorthEast()
  return lon >= sw.lng && lon <= ne.lng && lat >= sw.lat && lat <= ne.lat
}

/** Compute solar terminator polygon for day/night overlay */
function computeSolarTerminator(): GeoJSON.Feature<GeoJSON.Polygon> {
  const now = new Date()
  const dayOfYear = Math.floor(
    (now.getTime() - new Date(now.getFullYear(), 0, 0).getTime()) / 86400000
  )
  // Solar declination (approximate)
  const declination = -23.44 * Math.cos((2 * Math.PI / 365) * (dayOfYear + 10))
  const decRad = (declination * Math.PI) / 180

  // Hour angle of the sun
  const utcHours = now.getUTCHours() + now.getUTCMinutes() / 60 + now.getUTCSeconds() / 3600
  const solarNoonLng = (12 - utcHours) * 15

  const coords: [number, number][] = []

  // Terminator line
  for (let lng = -180; lng <= 180; lng += 2) {
    const lngRad = ((lng - solarNoonLng) * Math.PI) / 180
    const lat = (Math.atan(-Math.cos(lngRad) / Math.tan(decRad)) * 180) / Math.PI
    coords.push([lng, lat])
  }

  // Close the polygon on the night side
  // If declination > 0, night is on the south side; otherwise north
  const nightLat = declination > 0 ? -90 : 90
  coords.push([180, nightLat])
  coords.push([-180, nightLat])
  coords.push(coords[0]) // close ring

  return {
    type: 'Feature',
    properties: {},
    geometry: {
      type: 'Polygon',
      coordinates: [coords],
    },
  }
}

/** Format timestamp for popups */
function formatTime(ts: string | number): string {
  const d = typeof ts === 'number' ? new Date(ts) : new Date(ts)
  return d.toLocaleString('en-US', {
    month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
    hour12: false,
  })
}

// ─── GeoJSON Builders ─────────────────────────────────────────────

function buildEventsGeoJSON(events: SentinelEvent[], bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = []
  for (const evt of events) {
    if (evt.location.type !== 'Point') continue
    const [lon, lat] = evt.location.coordinates as number[]
    if (bounds && !inBounds(lon, lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [lon, lat] },
      properties: {
        id: evt.id,
        title: evt.title,
        category: evt.category,
        severity: evt.severity,
        color: SEVERITY_COLORS[evt.severity] || '#6b7280',
        occurred_at: evt.occurred_at,
        truth_score: evt.truth_score,
        source: evt.source,
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildAircraftGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData) return emptyFC()
  const all: Aircraft[] = [
    ...(liveData.commercial_flights || []),
    ...(liveData.military_flights || []),
    ...(liveData.private_flights || []),
  ]
  const features: GeoJSON.Feature[] = []
  for (const ac of all) {
    if (bounds && !inBounds(ac.lon, ac.lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [ac.lon, ac.lat] },
      properties: {
        callsign: ac.callsign || ac.icao,
        heading: ac.heading || 0,
        alt_ft: ac.alt_ft,
        speed_kts: ac.speed_kts,
        category: ac.category,
        color: AIRCRAFT_COLORS[ac.category] || '#06b6d4',
        squawk: ac.squawk || '',
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildShipsGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.ships) return emptyFC()
  const features: GeoJSON.Feature[] = []
  for (const v of liveData.ships) {
    if (bounds && !inBounds(v.lon, v.lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [v.lon, v.lat] },
      properties: {
        name: v.name || v.mmsi,
        ship_type: v.ship_type,
        speed: v.speed,
        course: v.course,
        destination: v.destination || '',
        flag: v.flag || '',
        color: SHIP_COLORS[v.ship_type] || '#6b7280',
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildSatellitesGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.satellites) return emptyFC()
  const features: GeoJSON.Feature[] = []
  for (const sat of liveData.satellites) {
    if (bounds && !inBounds(sat.lon, sat.lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [sat.lon, sat.lat] },
      properties: {
        name: sat.name,
        norad_id: sat.norad_id,
        alt_km: sat.alt_km,
        speed_kph: sat.speed_kph,
        mission_type: sat.mission_type,
        country: sat.country || '',
        color: SAT_COLORS[sat.mission_type] || '#22c55e',
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildEarthquakesGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.earthquakes) return emptyFC()
  const now = Date.now()
  const features: GeoJSON.Feature[] = []
  for (const eq of liveData.earthquakes) {
    if (bounds && !inBounds(eq.lon, eq.lat, bounds)) continue
    const isRecent = (now - eq.time) < 3600000
    let color = '#22c55e'
    if (eq.mag >= 7) color = '#ef4444'
    else if (eq.mag >= 5) color = '#f97316'
    else if (eq.mag >= 3) color = '#eab308'
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [eq.lon, eq.lat] },
      properties: {
        id: eq.id,
        mag: eq.mag,
        place: eq.place,
        depth_km: eq.depth_km,
        time: eq.time,
        url: eq.url,
        color,
        radius: Math.max(4, eq.mag * 3),
        isRecent: isRecent ? 1 : 0,
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildFiresGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.firms_fires) return emptyFC()
  const features: GeoJSON.Feature[] = []
  for (const f of liveData.firms_fires) {
    if (bounds && !inBounds(f.lon, f.lat, bounds)) continue
    let color = '#eab308'
    if (f.frp > 100) color = '#ef4444'
    else if (f.frp > 30) color = '#f97316'
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [f.lon, f.lat] },
      properties: {
        brightness: f.brightness,
        frp: f.frp,
        confidence: f.confidence,
        acq_date: f.acq_date,
        color,
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildConflictsGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.gdelt) return emptyFC()
  const features: GeoJSON.Feature[] = []
  for (const g of liveData.gdelt) {
    if (bounds && !inBounds(g.lon, g.lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [g.lon, g.lat] },
      properties: {
        title: g.title,
        tone: g.tone,
        url: g.url,
        domain: g.domain,
        date: g.date,
        size: Math.max(4, Math.abs(g.tone) * 1.5),
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildSigintGeoJSON(liveData: LiveData | null, bounds: maplibregl.LngLatBounds | null): GeoJSON.FeatureCollection {
  if (!liveData?.kiwisdr) return emptyFC()
  const features: GeoJSON.Feature[] = []
  for (const sdr of liveData.kiwisdr) {
    if (bounds && !inBounds(sdr.lon, sdr.lat, bounds)) continue
    features.push({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [sdr.lon, sdr.lat] },
      properties: {
        name: sdr.name,
        url: sdr.url,
        bands: sdr.bands || '',
        users_active: sdr.users_active ?? 0,
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

function buildCorrelationsGeoJSON(correlations: CorrelationFlash[]): GeoJSON.FeatureCollection {
  const features: GeoJSON.Feature[] = []
  for (const c of correlations) {
    // Build a circle polygon from center + radius
    const steps = 64
    const coords: [number, number][] = []
    const radiusDeg = c.radius_km / 111.32 // rough km to degree
    for (let i = 0; i <= steps; i++) {
      const angle = (i / steps) * 2 * Math.PI
      const lng = c.lon + radiusDeg * Math.cos(angle) / Math.cos((c.lat * Math.PI) / 180)
      const lat = c.lat + radiusDeg * Math.sin(angle)
      coords.push([lng, lat])
    }
    features.push({
      type: 'Feature',
      geometry: { type: 'Polygon', coordinates: [coords] },
      properties: {
        id: c.id,
        region_name: c.region_name,
        event_count: c.event_count,
        source_count: c.source_count,
        confirmed: c.confirmed ? 1 : 0,
        incident_name: c.incident_name || '',
        color: c.confirmed ? '#ef4444' : '#f59e0b',
      },
    })
  }
  return { type: 'FeatureCollection', features }
}

// ─── CSS Filter Overlays ──────────────────────────────────────────

function getCanvasFilter(style: MapStyleKey): string {
  switch (style) {
    case 'flir': return 'hue-rotate(180deg) invert(1)'
    case 'nvg': return 'saturate(0) brightness(0.8)'
    case 'crt': return 'saturate(0.5) contrast(1.2)'
    default: return 'none'
  }
}

// ─── Component ────────────────────────────────────────────────────

export default function MaplibreViewer({
  events,
  liveData,
  selectedEvent,
  onSelectEvent,
  flyTo,
  mapStyle,
  visibleLayers,
  correlations,
  onMouseMove,
}: MaplibreViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<maplibregl.Map | null>(null)
  const popupRef = useRef<maplibregl.Popup | null>(null)
  const sourcesReadyRef = useRef(false)
  const updateTimersRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({})
  const prevStyleRef = useRef<MapStyleKey>(mapStyle)
  const terminatorTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // ─── Debounced source update ────────────────────────────────────

  const scheduleSourceUpdate = useCallback((sourceId: string, data: GeoJSON.FeatureCollection) => {
    const map = mapRef.current
    if (!map || !sourcesReadyRef.current) return

    if (updateTimersRef.current[sourceId]) {
      clearTimeout(updateTimersRef.current[sourceId])
    }

    updateTimersRef.current[sourceId] = setTimeout(() => {
      delete updateTimersRef.current[sourceId]
      if (!mapRef.current) return
      const src = mapRef.current.getSource(sourceId) as maplibregl.GeoJSONSource | undefined
      if (src) {
        src.setData(data)
      }
    }, DEBOUNCE_MS)
  }, [])

  // ─── Initialize map ─────────────────────────────────────────────

  useEffect(() => {
    if (!containerRef.current) return

    const map = new maplibregl.Map({
      container: containerRef.current,
      style: MAP_STYLES[mapStyle].url,
      center: [0, 20],
      zoom: 2.5,
      maxZoom: 18,
      minZoom: 1.5,
      attributionControl: false,
      failIfMajorPerformanceCaveat: false,
    })

    map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left')
    map.addControl(new maplibregl.NavigationControl({ showCompass: true, showZoom: false }), 'top-right')

    mapRef.current = map

    map.on('load', () => {
      addAllSources(map)
      addAllLayers(map)
      sourcesReadyRef.current = true

      // Apply initial visibility
      applyVisibility(map, visibleLayers)

      // Initial terminator
      updateTerminator(map)

      // Update terminator every 60s
      terminatorTimerRef.current = setInterval(() => {
        if (mapRef.current) updateTerminator(mapRef.current)
      }, 60000)
    })

    map.on('mousemove', (e) => {
      onMouseMove?.([e.lngLat.lng, e.lngLat.lat])
    })

    map.on('error', (e) => {
      console.error('[MaplibreViewer] Map error:', e.error?.message || e)
    })

    return () => {
      sourcesReadyRef.current = false
      // Clear all pending debounce timers
      for (const t of Object.values(updateTimersRef.current)) clearTimeout(t)
      updateTimersRef.current = {}
      if (terminatorTimerRef.current) clearInterval(terminatorTimerRef.current)
      if (popupRef.current) popupRef.current.remove()
      map.remove()
      mapRef.current = null
    }
    // Only run on mount/unmount. Style changes handled separately.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ─── Style changes ──────────────────────────────────────────────

  useEffect(() => {
    const map = mapRef.current
    if (!map || mapStyle === prevStyleRef.current) return
    prevStyleRef.current = mapStyle

    sourcesReadyRef.current = false
    map.setStyle(MAP_STYLES[mapStyle].url)

    map.once('style.load', () => {
      addAllSources(map)
      addAllLayers(map)
      sourcesReadyRef.current = true
      applyVisibility(map, visibleLayers)
      updateTerminator(map)
      // Re-push current data
      pushAllData()
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mapStyle])

  // ─── FlyTo ──────────────────────────────────────────────────────

  useEffect(() => {
    if (!flyTo || !mapRef.current) return
    mapRef.current.flyTo({
      center: flyTo,
      zoom: 10,
      speed: 1.5,
      curve: 1.4,
    })
  }, [flyTo])

  // ─── Layer visibility ───────────────────────────────────────────

  useEffect(() => {
    if (!mapRef.current || !sourcesReadyRef.current) return
    applyVisibility(mapRef.current, visibleLayers)
  }, [visibleLayers])

  // ─── Data updates ───────────────────────────────────────────────

  const getBounds = useCallback(() => {
    if (!mapRef.current) return null
    return getBufferedBounds(mapRef.current)
  }, [])

  // Push all data to sources
  const pushAllData = useCallback(() => {
    const bounds = getBounds()
    scheduleSourceUpdate(SOURCE_IDS.events, buildEventsGeoJSON(events, bounds))
    scheduleSourceUpdate(SOURCE_IDS.aircraft, buildAircraftGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.ships, buildShipsGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.satellites, buildSatellitesGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.earthquakes, buildEarthquakesGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.fires, buildFiresGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.conflicts, buildConflictsGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.sigint, buildSigintGeoJSON(liveData, bounds))
    scheduleSourceUpdate(SOURCE_IDS.correlations, buildCorrelationsGeoJSON(correlations))
  }, [events, liveData, correlations, getBounds, scheduleSourceUpdate])

  useEffect(() => {
    if (!sourcesReadyRef.current) return
    pushAllData()
  }, [pushAllData])

  // Re-cull on move/zoom
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const onMoveEnd = debounce(() => {
      if (sourcesReadyRef.current) pushAllData()
    }, DEBOUNCE_MS)
    map.on('moveend', onMoveEnd)
    return () => {
      map.off('moveend', onMoveEnd)
      onMoveEnd.cancel()
    }
  }, [pushAllData])

  // ─── Interactions setup ─────────────────────────────────────────

  useEffect(() => {
    const map = mapRef.current
    if (!map) return

    const handleReady = () => {
      setupInteractions(map)
    }

    if (sourcesReadyRef.current) {
      setupInteractions(map)
    } else {
      map.once('style.load', handleReady)
    }

    return () => {
      map.off('style.load', handleReady)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onSelectEvent])

  // Setup click/hover interactions
  const setupInteractions = useCallback((map: maplibregl.Map) => {
    // Shared popup
    if (!popupRef.current) {
      popupRef.current = new maplibregl.Popup({
        closeButton: false,
        closeOnClick: false,
        maxWidth: '320px',
        className: 'sentinel-popup',
      })
    }
    const popup = popupRef.current

    // ── SENTINEL Events click ──
    map.on('click', LAYER_IDS.events, (e) => {
      if (!e.features?.length) return
      const props = e.features[0].properties
      if (props?.id) {
        const found = events.find(ev => ev.id === props.id)
        onSelectEvent(found || null)
      }
    })

    // ── SENTINEL Events hover ──
    map.on('mouseenter', LAYER_IDS.events, (e) => {
      map.getCanvas().style.cursor = 'pointer'
      if (!e.features?.length) return
      const f = e.features[0]
      const coords = (f.geometry as GeoJSON.Point).coordinates.slice() as [number, number]
      const p = f.properties!
      popup.setLngLat(coords).setHTML(`
        <div class="font-mono text-xs leading-tight">
          <div class="font-bold text-white mb-1">${escapeHtml(p.title)}</div>
          <div><span class="text-neutral-400">CAT:</span> ${escapeHtml(p.category)}</div>
          <div><span class="text-neutral-400">SEV:</span> <span style="color:${p.color}">${p.severity?.toUpperCase()}</span></div>
          <div><span class="text-neutral-400">SRC:</span> ${escapeHtml(p.source)}</div>
          <div><span class="text-neutral-400">TIME:</span> ${formatTime(p.occurred_at)}</div>
        </div>
      `).addTo(map)
    })
    map.on('mouseleave', LAYER_IDS.events, () => {
      map.getCanvas().style.cursor = ''
      popup.remove()
    })

    // ── Aircraft hover ──
    setupHoverPopup(map, popup, LAYER_IDS.aircraft, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold" style="color:${p.color}">${escapeHtml(p.callsign)}</div>
        <div>ALT: ${Number(p.alt_ft).toLocaleString()} ft | SPD: ${p.speed_kts} kts</div>
        <div>TYPE: ${p.category}${p.squawk ? ` | SQK: ${p.squawk}` : ''}</div>
      </div>
    `)

    // ── Ships hover ──
    setupHoverPopup(map, popup, LAYER_IDS.ships, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold" style="color:${p.color}">${escapeHtml(p.name)}</div>
        <div>TYPE: ${p.ship_type} | SPD: ${p.speed} kts</div>
        ${p.destination ? `<div>DEST: ${escapeHtml(p.destination)}</div>` : ''}
        ${p.flag ? `<div>FLAG: ${p.flag}</div>` : ''}
      </div>
    `)

    // ── Satellites hover ──
    setupHoverPopup(map, popup, LAYER_IDS.satellites, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold" style="color:${p.color}">${escapeHtml(p.name)}</div>
        <div>ALT: ${Number(p.alt_km).toLocaleString()} km | NORAD: ${p.norad_id}</div>
        <div>MISSION: ${p.mission_type}${p.country ? ` | ${p.country}` : ''}</div>
      </div>
    `)

    // ── Earthquakes hover ──
    setupHoverPopup(map, popup, LAYER_IDS.earthquakes, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold" style="color:${p.color}">M${p.mag} EARTHQUAKE</div>
        <div>${escapeHtml(p.place)}</div>
        <div>DEPTH: ${p.depth_km} km | ${formatTime(Number(p.time))}</div>
      </div>
    `)

    // ── Fires hover ──
    setupHoverPopup(map, popup, LAYER_IDS.fires, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold" style="color:${p.color}">FIRE DETECTION</div>
        <div>FRP: ${p.frp} MW | BRIGHT: ${p.brightness}</div>
        <div>CONF: ${p.confidence} | ${p.acq_date}</div>
      </div>
    `)

    // ── Conflicts hover ──
    setupHoverPopup(map, popup, LAYER_IDS.conflicts, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold text-red-400">GDELT CONFLICT</div>
        <div class="truncate max-w-[280px]">${escapeHtml(p.title)}</div>
        <div>TONE: ${Number(p.tone).toFixed(1)} | ${escapeHtml(p.domain)}</div>
      </div>
    `)

    // ── KiwiSDR click ──
    map.on('click', LAYER_IDS.sigint, (e) => {
      if (!e.features?.length) return
      const url = e.features[0].properties?.url
      if (url) window.open(url, '_blank', 'noopener')
    })
    setupHoverPopup(map, popup, LAYER_IDS.sigint, (p) => `
      <div class="font-mono text-xs">
        <div class="font-bold text-green-400">${escapeHtml(p.name)}</div>
        ${p.bands ? `<div>BANDS: ${escapeHtml(p.bands)}</div>` : ''}
        <div>USERS: ${p.users_active} | Click to open</div>
      </div>
    `)

    // ── Correlations hover ──
    map.on('mouseenter', LAYER_IDS.correlations, (e) => {
      map.getCanvas().style.cursor = 'pointer'
      if (!e.features?.length) return
      const p = e.features[0].properties!
      const coords = e.lngLat
      popup.setLngLat(coords).setHTML(`
        <div class="font-mono text-xs">
          <div class="font-bold" style="color:${p.color}">
            ${p.confirmed ? 'CONFIRMED' : 'SUSPECTED'} CORRELATION
          </div>
          <div>${escapeHtml(p.region_name)}</div>
          <div>EVENTS: ${p.event_count} | SOURCES: ${p.source_count}</div>
          ${p.incident_name ? `<div>INCIDENT: ${escapeHtml(p.incident_name)}</div>` : ''}
        </div>
      `).addTo(map)
    })
    map.on('mouseleave', LAYER_IDS.correlations, () => {
      map.getCanvas().style.cursor = ''
      popup.remove()
    })

    // Click on empty space deselects
    map.on('click', (e) => {
      const features = map.queryRenderedFeatures(e.point, {
        layers: [LAYER_IDS.events],
      })
      if (!features.length) {
        onSelectEvent(null)
      }
    })
  }, [events, onSelectEvent])

  // Helper for hover popup pattern
  function setupHoverPopup(
    map: maplibregl.Map,
    popup: maplibregl.Popup,
    layerId: string,
    htmlBuilder: (props: Record<string, string | number>) => string,
  ) {
    map.on('mouseenter', layerId, (e) => {
      map.getCanvas().style.cursor = 'pointer'
      if (!e.features?.length) return
      const f = e.features[0]
      const coords = (f.geometry as GeoJSON.Point).coordinates.slice() as [number, number]
      popup.setLngLat(coords).setHTML(htmlBuilder(f.properties as Record<string, string | number>)).addTo(map)
    })
    map.on('mouseleave', layerId, () => {
      map.getCanvas().style.cursor = ''
      popup.remove()
    })
  }

  // ─── Selected event highlight ───────────────────────────────────

  useEffect(() => {
    const map = mapRef.current
    if (!map || !sourcesReadyRef.current) return
    // Pan to selected event
    if (selectedEvent?.location.type === 'Point') {
      const [lon, lat] = selectedEvent.location.coordinates as number[]
      map.easeTo({ center: [lon, lat], duration: 600 })
    }
  }, [selectedEvent])

  // ─── Earthquake pulse animation ─────────────────────────────────

  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    let animFrame: number
    let start: number | null = null

    const animate = (ts: number) => {
      if (!start) start = ts
      const elapsed = ts - start
      // Pulsing radius multiplier between 1 and 2 over 2 seconds
      const pulse = 1 + Math.sin((elapsed / 1000) * Math.PI) * 0.5

      if (map.getLayer(LAYER_IDS.earthquakesPulse)) {
        map.setPaintProperty(LAYER_IDS.earthquakesPulse, 'circle-radius', [
          '*', ['get', 'radius'], pulse,
        ])
        map.setPaintProperty(LAYER_IDS.earthquakesPulse, 'circle-opacity', [
          'interpolate', ['linear'], ['literal', pulse],
          1, 0.6, 1.5, 0.1,
        ])
      }
      animFrame = requestAnimationFrame(animate)
    }

    animFrame = requestAnimationFrame(animate)
    return () => cancelAnimationFrame(animFrame)
  }, [])

  // ─── Overlay styles (computed from mapStyle) ────────────────────

  const overlayStyle = useMemo(() => {
    switch (mapStyle) {
      case 'flir':
        return { background: 'rgba(255, 140, 0, 0.06)', pointerEvents: 'none' as const }
      case 'nvg':
        return { background: 'rgba(0, 255, 0, 0.05)', pointerEvents: 'none' as const }
      case 'crt':
        return { background: 'rgba(0, 255, 50, 0.03)', pointerEvents: 'none' as const }
      default:
        return null
    }
  }, [mapStyle])

  const canvasFilter = useMemo(() => getCanvasFilter(mapStyle), [mapStyle])

  // Apply CSS filter to map canvas
  useEffect(() => {
    const map = mapRef.current
    if (!map) return
    const canvas = map.getCanvas()
    if (canvas) {
      canvas.style.filter = canvasFilter
    }
  }, [canvasFilter])

  // ─── Render ─────────────────────────────────────────────────────

  return (
    <div className="relative w-full h-full overflow-hidden">
      <div ref={containerRef} className="absolute inset-0" />

      {/* Color tint overlay for FLIR/NVG/CRT */}
      {overlayStyle && (
        <div className="absolute inset-0" style={overlayStyle} />
      )}

      {/* CRT scanlines */}
      {mapStyle === 'crt' && (
        <div
          className="absolute inset-0 pointer-events-none"
          style={{
            backgroundImage: 'repeating-linear-gradient(0deg, rgba(0,0,0,0.15) 0px, transparent 1px, transparent 3px)',
            backgroundSize: '100% 3px',
            mixBlendMode: 'multiply',
          }}
        />
      )}
    </div>
  )
}

// ─── Source & Layer Setup ──────────────────────────────────────────

function addAllSources(map: maplibregl.Map) {
  const sources: [string, GeoJSON.FeatureCollection][] = [
    [SOURCE_IDS.terminator, emptyFC()],
    [SOURCE_IDS.correlations, emptyFC()],
    [SOURCE_IDS.fires, emptyFC()],
    [SOURCE_IDS.earthquakes, emptyFC()],
    [SOURCE_IDS.conflicts, emptyFC()],
    [SOURCE_IDS.ships, emptyFC()],
    [SOURCE_IDS.aircraft, emptyFC()],
    [SOURCE_IDS.satellites, emptyFC()],
    [SOURCE_IDS.sigint, emptyFC()],
    [SOURCE_IDS.events, emptyFC()],
  ]

  for (const [id, data] of sources) {
    if (!map.getSource(id)) {
      map.addSource(id, { type: 'geojson', data })
    }
  }
}

function addAllLayers(map: maplibregl.Map) {
  // Order matters: bottom layers first

  // ── Solar terminator ──
  if (!map.getLayer(LAYER_IDS.terminator)) {
    map.addLayer({
      id: LAYER_IDS.terminator,
      type: 'fill',
      source: SOURCE_IDS.terminator,
      paint: {
        'fill-color': '#000000',
        'fill-opacity': 0.3,
      },
    })
  }

  // ── Correlation circles ──
  if (!map.getLayer(LAYER_IDS.correlations)) {
    map.addLayer({
      id: LAYER_IDS.correlations,
      type: 'line',
      source: SOURCE_IDS.correlations,
      paint: {
        'line-color': ['get', 'color'],
        'line-width': 2,
        'line-dasharray': [4, 4],
        'line-opacity': 0.7,
      },
    })
  }

  // ── Fires ──
  if (!map.getLayer(LAYER_IDS.fires)) {
    map.addLayer({
      id: LAYER_IDS.fires,
      type: 'circle',
      source: SOURCE_IDS.fires,
      paint: {
        'circle-radius': 3,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.8,
        'circle-blur': 0.3,
      },
    })
  }

  // ── Earthquakes (base) ──
  if (!map.getLayer(LAYER_IDS.earthquakes)) {
    map.addLayer({
      id: LAYER_IDS.earthquakes,
      type: 'circle',
      source: SOURCE_IDS.earthquakes,
      paint: {
        'circle-radius': ['get', 'radius'],
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.75,
        'circle-stroke-color': '#ffffff',
        'circle-stroke-width': 1,
        'circle-stroke-opacity': 0.5,
      },
    })
  }

  // ── Earthquakes (pulse ring for recent) ──
  if (!map.getLayer(LAYER_IDS.earthquakesPulse)) {
    map.addLayer({
      id: LAYER_IDS.earthquakesPulse,
      type: 'circle',
      source: SOURCE_IDS.earthquakes,
      filter: ['==', ['get', 'isRecent'], 1],
      paint: {
        'circle-radius': ['get', 'radius'],
        'circle-color': 'transparent',
        'circle-stroke-color': ['get', 'color'],
        'circle-stroke-width': 2,
        'circle-opacity': 0.4,
      },
    })
  }

  // ── GDELT Conflicts ──
  if (!map.getLayer(LAYER_IDS.conflicts)) {
    map.addLayer({
      id: LAYER_IDS.conflicts,
      type: 'circle',
      source: SOURCE_IDS.conflicts,
      paint: {
        'circle-radius': ['get', 'size'],
        'circle-color': '#ef4444',
        'circle-opacity': 0.6,
        'circle-stroke-color': '#fca5a5',
        'circle-stroke-width': 1,
      },
    })
  }

  // ── Ships (diamonds via rotation) ──
  if (!map.getLayer(LAYER_IDS.ships)) {
    map.addLayer({
      id: LAYER_IDS.ships,
      type: 'circle',
      source: SOURCE_IDS.ships,
      paint: {
        'circle-radius': 4,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.85,
        'circle-stroke-color': '#000000',
        'circle-stroke-width': 1,
      },
    })
  }

  // ── Aircraft ──
  if (!map.getLayer(LAYER_IDS.aircraft)) {
    map.addLayer({
      id: LAYER_IDS.aircraft,
      type: 'circle',
      source: SOURCE_IDS.aircraft,
      paint: {
        'circle-radius': 3,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.9,
        'circle-stroke-color': '#000000',
        'circle-stroke-width': 0.5,
      },
    })
  }

  // ── Satellites ──
  if (!map.getLayer(LAYER_IDS.satellites)) {
    map.addLayer({
      id: LAYER_IDS.satellites,
      type: 'circle',
      source: SOURCE_IDS.satellites,
      paint: {
        'circle-radius': 2,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.8,
      },
    })
  }

  // ── KiwiSDR ──
  if (!map.getLayer(LAYER_IDS.sigint)) {
    map.addLayer({
      id: LAYER_IDS.sigint,
      type: 'circle',
      source: SOURCE_IDS.sigint,
      paint: {
        'circle-radius': 5,
        'circle-color': '#22c55e',
        'circle-opacity': 0.8,
        'circle-stroke-color': '#86efac',
        'circle-stroke-width': 2,
      },
    })
  }

  // ── SENTINEL Events glow ──
  if (!map.getLayer(LAYER_IDS.eventsGlow)) {
    map.addLayer({
      id: LAYER_IDS.eventsGlow,
      type: 'circle',
      source: SOURCE_IDS.events,
      paint: {
        'circle-radius': 12,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.15,
        'circle-blur': 1,
      },
    })
  }

  // ── SENTINEL Events ──
  if (!map.getLayer(LAYER_IDS.events)) {
    map.addLayer({
      id: LAYER_IDS.events,
      type: 'circle',
      source: SOURCE_IDS.events,
      paint: {
        'circle-radius': 6,
        'circle-color': ['get', 'color'],
        'circle-opacity': 0.9,
        'circle-stroke-color': '#ffffff',
        'circle-stroke-width': 1.5,
        'circle-stroke-opacity': 0.6,
      },
    })
  }
}

function applyVisibility(map: maplibregl.Map, visibleLayers: Set<string>) {
  for (const [key, layerIds] of Object.entries(LAYER_KEY_TO_IDS)) {
    const vis = visibleLayers.has(key) ? 'visible' : 'none'
    for (const id of layerIds) {
      if (map.getLayer(id)) {
        map.setLayoutProperty(id, 'visibility', vis)
      }
    }
  }
}

function updateTerminator(map: maplibregl.Map) {
  const src = map.getSource(SOURCE_IDS.terminator) as maplibregl.GeoJSONSource | undefined
  if (src) {
    src.setData({
      type: 'FeatureCollection',
      features: [computeSolarTerminator()],
    })
  }
}

function escapeHtml(str: string | undefined | null): string {
  if (!str) return ''
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
