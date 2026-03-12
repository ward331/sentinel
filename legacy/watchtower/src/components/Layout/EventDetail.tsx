import type { SentinelEvent } from '../../types/sentinel'
import { X, MapPin, Clock, Tag, Gauge } from 'lucide-react'

const SEVERITY_BG: Record<string, string> = {
  critical: 'bg-red-900/30 border-red-800 text-red-300',
  high: 'bg-orange-900/30 border-orange-800 text-orange-300',
  medium: 'bg-yellow-900/30 border-yellow-800 text-yellow-300',
  low: 'bg-green-900/30 border-green-800 text-green-300',
}

interface Props {
  event: SentinelEvent
  onClose: () => void
}

export function EventDetail({ event, onClose }: Props) {
  const coords = event.location?.type === 'Point' && Array.isArray(event.location.coordinates)
    ? event.location.coordinates as number[]
    : null

  return (
    <div className="absolute bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-800 max-h-[40%] overflow-y-auto z-[1000]">
      <div className="p-4">
        <div className="flex items-start justify-between mb-3">
          <div>
            <h3 className="text-base font-semibold text-white">{event.title}</h3>
            <p className="text-xs text-gray-500 mt-0.5">{event.source} &middot; {event.id.slice(0, 8)}</p>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-gray-800 rounded">
            <X className="w-4 h-4 text-gray-400" />
          </button>
        </div>

        <div className="flex flex-wrap gap-2 mb-3">
          {event.severity && (
            <span className={`text-xs px-2 py-0.5 rounded border ${SEVERITY_BG[event.severity] || ''}`}>
              {event.severity.toUpperCase()}
            </span>
          )}
          {event.category && (
            <span className="text-xs px-2 py-0.5 rounded bg-gray-800 text-gray-400 flex items-center gap-1">
              <Tag className="w-3 h-3" />{event.category}
            </span>
          )}
          {event.magnitude > 0 && (
            <span className="text-xs px-2 py-0.5 rounded bg-gray-800 text-gray-400 flex items-center gap-1">
              <Gauge className="w-3 h-3" />M{event.magnitude.toFixed(1)}
            </span>
          )}
        </div>

        {event.description && (
          <p className="text-sm text-gray-300 mb-3">{event.description}</p>
        )}

        <div className="grid grid-cols-2 gap-2 text-xs">
          <div className="flex items-center gap-1.5 text-gray-400">
            <Clock className="w-3.5 h-3.5" />
            {new Date(event.occurred_at).toLocaleString()}
          </div>
          {coords && (
            <div className="flex items-center gap-1.5 text-gray-400">
              <MapPin className="w-3.5 h-3.5" />
              {coords[1].toFixed(3)}, {coords[0].toFixed(3)}
            </div>
          )}
        </div>

        {event.badges?.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-3">
            {event.badges.map((b, i) => (
              <span key={i} className="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-500">
                {b.label}
              </span>
            ))}
          </div>
        )}

        {event.metadata && Object.keys(event.metadata).length > 0 && (
          <div className="mt-3 border-t border-gray-800 pt-3">
            <p className="text-xs font-medium text-gray-400 mb-1">Metadata</p>
            <div className="grid grid-cols-2 gap-1">
              {Object.entries(event.metadata).map(([k, v]) => (
                <div key={k} className="text-xs">
                  <span className="text-gray-500">{k}:</span>{' '}
                  <span className="text-gray-300">{v}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
