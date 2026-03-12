import { Shield, Settings, Activity, AlertTriangle, Eye } from 'lucide-react'

export type View = 'map' | 'health' | 'alerts'

interface Props {
  view: View
  onViewChange: (view: View) => void
  onOpenSettings: () => void
  connected: boolean
  eventCount: number
}

export function Header({ view, onViewChange, onOpenSettings, connected, eventCount }: Props) {
  return (
    <header className="h-12 bg-gray-900 border-b border-gray-800 flex items-center px-4 justify-between shrink-0">
      <div className="flex items-center gap-3">
        <Shield className="w-5 h-5 text-emerald-400" />
        <span className="font-bold text-white tracking-tight">WATCHTOWER</span>
        <span className="text-xs text-gray-500 hidden sm:inline">SENTINEL V2</span>
      </div>

      <nav className="flex items-center gap-1">
        <NavBtn active={view === 'map'} onClick={() => onViewChange('map')}>
          <Eye className="w-4 h-4" />
          <span className="hidden sm:inline">Map</span>
        </NavBtn>
        <NavBtn active={view === 'health'} onClick={() => onViewChange('health')}>
          <Activity className="w-4 h-4" />
          <span className="hidden sm:inline">Health</span>
        </NavBtn>
        <NavBtn active={view === 'alerts'} onClick={() => onViewChange('alerts')}>
          <AlertTriangle className="w-4 h-4" />
          <span className="hidden sm:inline">Alerts</span>
        </NavBtn>
      </nav>

      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span className={`w-2 h-2 rounded-full ${connected ? 'bg-emerald-400' : 'bg-red-400'}`} />
          <span className="text-xs text-gray-500">{eventCount}</span>
        </div>
        <button
          onClick={onOpenSettings}
          className="p-1.5 rounded hover:bg-gray-800 transition-colors"
        >
          <Settings className="w-4 h-4 text-gray-400" />
        </button>
      </div>
    </header>
  )
}

function NavBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-3 py-1.5 rounded text-sm transition-colors ${
        active ? 'bg-gray-800 text-emerald-400' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
      }`}
    >
      {children}
    </button>
  )
}
