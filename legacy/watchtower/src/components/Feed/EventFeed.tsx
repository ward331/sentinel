import type { SentinelEvent } from '../../types/sentinel'
import { EventCard } from './EventCard'

interface Props {
  events: SentinelEvent[]
  onSelectEvent?: (event: SentinelEvent) => void
  loading?: boolean
}

export function EventFeed({ events, onSelectEvent, loading }: Props) {
  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b border-gray-800 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Live Feed</h2>
        <span className="text-xs text-gray-500">{events.length} events</span>
      </div>
      <div className="flex-1 overflow-y-auto event-feed">
        {loading && events.length === 0 && (
          <div className="p-4 text-center text-gray-500 text-sm">Loading events...</div>
        )}
        {!loading && events.length === 0 && (
          <div className="p-4 text-center text-gray-500 text-sm">No events match your filters</div>
        )}
        {events.map(event => (
          <EventCard
            key={event.id}
            event={event}
            onClick={() => onSelectEvent?.(event)}
          />
        ))}
      </div>
    </div>
  )
}
