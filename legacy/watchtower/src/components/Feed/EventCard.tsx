import type { SentinelEvent } from '../../types/sentinel'

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-500',
  high: 'bg-orange-500',
  medium: 'bg-yellow-500',
  low: 'bg-green-500',
}

const SEVERITY_BORDER: Record<string, string> = {
  critical: 'border-l-red-500',
  high: 'border-l-orange-500',
  medium: 'border-l-yellow-500',
  low: 'border-l-green-500',
}

function timeAgo(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

interface Props {
  event: SentinelEvent
  onClick: () => void
}

export function EventCard({ event, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-4 py-3 border-l-3 ${SEVERITY_BORDER[event.severity] || 'border-l-gray-600'} border-b border-b-gray-800/50 hover:bg-gray-800/50 transition-colors`}
    >
      <div className="flex items-start gap-2">
        <span className={`mt-1.5 w-2 h-2 rounded-full shrink-0 ${SEVERITY_COLORS[event.severity] || 'bg-gray-500'} ${event.severity === 'critical' ? 'pulse-critical' : ''}`} />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-gray-200 truncate">{event.title}</p>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-xs text-gray-500">{event.source}</span>
            {event.category && (
              <>
                <span className="text-gray-700">&middot;</span>
                <span className="text-xs text-gray-500">{event.category}</span>
              </>
            )}
            {event.magnitude > 0 && (
              <>
                <span className="text-gray-700">&middot;</span>
                <span className="text-xs font-mono text-gray-400">M{event.magnitude.toFixed(1)}</span>
              </>
            )}
          </div>
        </div>
        <span className="text-xs text-gray-600 shrink-0">{timeAgo(event.occurred_at)}</span>
      </div>
    </button>
  )
}
