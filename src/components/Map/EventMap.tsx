import { MapContainer, TileLayer, CircleMarker, Popup, useMap } from 'react-leaflet'
import { memo, useEffect, useRef } from 'react'
import type { SentinelEvent } from '../../types/sentinel'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'

const SEVERITY_COLORS: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#22c55e',
}

const SEVERITY_RADIUS: Record<string, number> = {
  critical: 10,
  high: 8,
  medium: 6,
  low: 5,
}

function getCoords(event: SentinelEvent): [number, number] | null {
  if (!event.location?.coordinates) return null
  const c = event.location.coordinates
  if (event.location.type === 'Point' && Array.isArray(c) && c.length >= 2) {
    return [c[1] as number, c[0] as number]
  }
  if (event.location.type === 'Polygon' && Array.isArray(c) && c.length > 0) {
    const ring = c as number[][]
    if (ring.length > 0 && Array.isArray(ring[0])) {
      let sumLat = 0, sumLon = 0
      for (const pt of ring) {
        sumLon += pt[0]; sumLat += pt[1]
      }
      return [sumLat / ring.length, sumLon / ring.length]
    }
  }
  return null
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleString()
  } catch {
    return ts
  }
}

// Manages markers imperatively via Leaflet API to avoid React re-renders
function MarkerLayer({ events, onSelectEvent }: { events: SentinelEvent[], onSelectEvent?: (e: SentinelEvent) => void }) {
  const map = useMap()
  const layerRef = useRef<L.LayerGroup>(L.layerGroup())
  const onSelectRef = useRef(onSelectEvent)
  onSelectRef.current = onSelectEvent

  useEffect(() => {
    layerRef.current.addTo(map)
    return () => { layerRef.current.remove() }
  }, [map])

  useEffect(() => {
    const group = layerRef.current
    group.clearLayers()

    const items = events.slice(0, 500)
    for (const event of items) {
      const coords = getCoords(event)
      if (!coords) continue

      const color = SEVERITY_COLORS[event.severity] || '#666'
      const radius = SEVERITY_RADIUS[event.severity] || 5

      const marker = L.circleMarker(coords, {
        radius,
        color,
        fillColor: color,
        fillOpacity: 0.7,
        weight: 2,
      })

      const mag = event.magnitude > 0 ? `<p style="font-size:11px;font-family:monospace">Magnitude: ${event.magnitude.toFixed(1)}</p>` : ''
      const desc = event.description ? `<p style="font-size:11px;color:#ccc;margin-top:4px">${event.description.slice(0, 150)}</p>` : ''

      marker.bindPopup(`
        <div style="min-width:200px">
          <div style="display:flex;align-items:center;gap:6px;margin-bottom:4px">
            <span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${color}"></span>
            <strong style="font-size:13px">${event.title}</strong>
          </div>
          <p style="font-size:11px;color:#999">${event.source} · ${event.category}</p>
          ${mag}
          <p style="font-size:11px;color:#777;margin-top:4px">${formatTime(event.occurred_at)}</p>
          ${desc}
        </div>
      `)

      marker.on('click', () => onSelectRef.current?.(event))
      group.addLayer(marker)
    }
  }, [events])

  return null
}

function InvalidateSize() {
  const map = useMap()
  useEffect(() => {
    const timer = setTimeout(() => map.invalidateSize(), 200)
    return () => clearTimeout(timer)
  }, [map])
  return null
}

interface Props {
  events: SentinelEvent[]
  onSelectEvent?: (event: SentinelEvent) => void
}

// memo prevents MapContainer from ever re-rendering
export const EventMap = memo(function EventMap({ events, onSelectEvent }: Props) {
  return (
    <div style={{ position: 'absolute', inset: 0 }}>
      <MapContainer
        center={[20, 0]}
        zoom={3}
        style={{ width: '100%', height: '100%' }}
        zoomControl={true}
      >
        <TileLayer
          url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
          attribution='&copy; <a href="https://openstreetmap.org/">OpenStreetMap</a>'
          maxZoom={19}
        />
        <InvalidateSize />
        <MarkerLayer events={events} onSelectEvent={onSelectEvent} />
      </MapContainer>
    </div>
  )
})
