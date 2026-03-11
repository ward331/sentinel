import { Shield, ShieldAlert, ShieldOff } from 'lucide-react'

interface StatusBarProps {
  health: 'operational' | 'degraded' | 'offline'
  mouseCoords: [number, number] | null
  sourceCounts: Record<string, number>
}

const HEALTH_CONFIG = {
  operational: {
    label: 'OPERATIONAL',
    color: 'text-green-500',
    dotColor: 'bg-green-500',
    icon: Shield,
  },
  degraded: {
    label: 'DEGRADED',
    color: 'text-yellow-500',
    dotColor: 'bg-yellow-500',
    icon: ShieldAlert,
  },
  offline: {
    label: 'OFFLINE',
    color: 'text-red-500',
    dotColor: 'bg-red-500',
    icon: ShieldOff,
  },
}

function formatCoord(val: number, pos: string, neg: string): string {
  const abs = Math.abs(val)
  const deg = Math.floor(abs)
  const min = ((abs - deg) * 60).toFixed(3)
  return `${deg}°${min}'${val >= 0 ? pos : neg}`
}

function formatUtcTime(): string {
  const now = new Date()
  return now.toISOString().slice(11, 19) + 'Z'
}

function formatLocalTime(): string {
  const now = new Date()
  return now.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

export default function StatusBar({ health, mouseCoords, sourceCounts }: StatusBarProps) {
  const cfg = HEALTH_CONFIG[health]
  const Icon = cfg.icon
  const totalSources = Object.keys(sourceCounts).length
  const totalCount = Object.values(sourceCounts).reduce((a, b) => a + b, 0)

  return (
    <div className="absolute bottom-0 left-0 right-0 h-6 bg-gray-950/95 border-t border-gray-800 flex items-center justify-between px-3 z-30 backdrop-blur-sm">
      {/* Left: System status */}
      <div className="flex items-center gap-2">
        <Icon size={11} className={cfg.color} />
        <span className={`text-[10px] font-mono tracking-wider ${cfg.color}`}>
          SENTINEL V4
        </span>
        <span className="text-[10px] font-mono text-gray-700">|</span>
        <div className="flex items-center gap-1.5">
          <div
            className={`w-1.5 h-1.5 rounded-full ${cfg.dotColor}`}
            style={{
              boxShadow:
                health === 'operational'
                  ? '0 0 4px rgba(34,197,94,0.5)'
                  : health === 'degraded'
                  ? '0 0 4px rgba(234,179,8,0.5)'
                  : '0 0 4px rgba(239,68,68,0.5)',
            }}
          />
          <span className={`text-[10px] font-mono tracking-wider ${cfg.color}`}>
            {cfg.label}
          </span>
        </div>
      </div>

      {/* Center: Mouse coordinates */}
      <div className="text-[10px] font-mono text-gray-500">
        {mouseCoords ? (
          <>
            <span>{formatCoord(mouseCoords[1], 'N', 'S')}</span>
            <span className="text-gray-700 mx-1">|</span>
            <span>{formatCoord(mouseCoords[0], 'E', 'W')}</span>
          </>
        ) : (
          <span className="text-gray-700">-- -- --</span>
        )}
      </div>

      {/* Right: Time + sources */}
      <div className="flex items-center gap-3">
        <span className="text-[10px] font-mono text-gray-600">
          {totalSources} SRC / {totalCount > 9999 ? `${(totalCount / 1000).toFixed(1)}k` : totalCount} OBJ
        </span>
        <span className="text-gray-800">|</span>
        <span className="text-[10px] font-mono text-cyan-600">{formatUtcTime()}</span>
        <span className="text-[10px] font-mono text-gray-600">{formatLocalTime()}</span>
      </div>
    </div>
  )
}
