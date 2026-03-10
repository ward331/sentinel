import { MapContainer, TileLayer, CircleMarker, Popup, useMap } from 'react-leaflet'
import { useEffect } from 'react'
import type { SentinelEvent } from '../../types/sentinel'
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
    return [c[1] as number, c[0] as number] // [lat, lon]
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

function AutoFit({ events }: { events: SentinelEvent[] }) {
  const map = useMap()
  useEffect(() => {
    if (events.length === 0) return
    const critical = events.filter(e => e.severity === 'critical')
    if (critical.length > 0) {
      const c = getCoords(critical[0])
      if (c) map.flyTo(c, 6, { duration: 1 })
    }
  }, [events.length])
  return null
}

interface Props {
  events: SentinelEvent[]
  onSelectEvent?: (event: SentinelEvent) => void
}

export function EventMap({ events, onSelectEvent }: Props) {
  const markers = events
    .map(e => ({ event: e, coords: getCoords(e) }))
    .filter((m): m is { event: SentinelEvent; coords: [number, number] } => m.coords !== null)

  return (
    <MapContainer
      center={[20, 0]}
      zoom={3}
      className="h-full w-full"
      zoomControl={true}
    >
      <TileLayer
        url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
        attribution='&copy; <a href="https://carto.com/">CARTO</a>'
      />
      <AutoFit events={events} />
      {markers.map(({ event, coords }) => (
        <CircleMarker
          key={event.id}
          center={coords}
          radius={SEVERITY_RADIUS[event.severity] || 5}
          pathOptions={{
            color: SEVERITY_COLORS[event.severity] || '#666',
            fillColor: SEVERITY_COLORS[event.severity] || '#666',
            fillOpacity: 0.7,
            weight: 2,
          }}
          eventHandlers={{
            click: () => onSelectEvent?.(event),
          }}
        >
          <Popup>
            <div className="min-w-[200px]">
              <div className="flex items-center gap-2 mb-1">
                <span
                  className="inline-block w-2.5 h-2.5 rounded-full"
                  style={{ background: SEVERITY_COLORS[event.severity] || '#666' }}
                />
                <strong className="text-sm">{event.title}</strong>
              </div>
              <p className="text-xs text-gray-400 mb-1">{event.source} &middot; {event.category}</p>
              {event.magnitude > 0 && (
                <p className="text-xs font-mono">Magnitude: {event.magnitude.toFixed(1)}</p>
              )}
              <p className="text-xs text-gray-500 mt-1">{formatTime(event.occurred_at)}</p>
              {event.description && (
                <p className="text-xs text-gray-300 mt-1 line-clamp-3">{event.description}</p>
              )}
            </div>
          </Popup>
        </CircleMarker>
      ))}
    </MapContainer>
  )
}
