import { Shield, Settings, Activity, AlertTriangle, Eye, Radio, TrendingUp, BookOpen } from 'lucide-react'

export type View = 'map' | 'intel' | 'financial' | 'health' | 'alerts' | 'osint' | 'settings'

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
        <span className="text-xs text-gray-500 hidden sm:inline">SENTINEL V3</span>
      </div>

      <nav className="flex items-center gap-0.5 overflow-x-auto">
        <NavBtn active={view === 'map'} onClick={() => onViewChange('map')}>
          <Eye className="w-4 h-4" />
          <span className="hidden md:inline">Map</span>
        </NavBtn>
        <NavBtn active={view === 'intel'} onClick={() => onViewChange('intel')}>
          <Radio className="w-4 h-4" />
          <span className="hidden md:inline">Intel</span>
        </NavBtn>
        <NavBtn active={view === 'financial'} onClick={() => onViewChange('financial')}>
          <TrendingUp className="w-4 h-4" />
          <span className="hidden md:inline">Markets</span>
        </NavBtn>
        <NavBtn active={view === 'health'} onClick={() => onViewChange('health')}>
          <Activity className="w-4 h-4" />
          <span className="hidden md:inline">Health</span>
        </NavBtn>
        <NavBtn active={view === 'alerts'} onClick={() => onViewChange('alerts')}>
          <AlertTriangle className="w-4 h-4" />
          <span className="hidden md:inline">Alerts</span>
        </NavBtn>
        <NavBtn active={view === 'osint'} onClick={() => onViewChange('osint')}>
          <BookOpen className="w-4 h-4" />
          <span className="hidden md:inline">OSINT</span>
        </NavBtn>
      </nav>

      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span className={`w-2 h-2 rounded-full ${connected ? 'bg-emerald-400' : 'bg-red-400'}`} />
          <span className="text-xs text-gray-500">{eventCount}</span>
        </div>
        <button
          onClick={() => onOpenSettings()}
          className={`p-1.5 rounded transition-colors ${view === 'settings' ? 'bg-gray-700 text-emerald-400' : 'hover:bg-gray-800 text-gray-400'}`}
        >
          <Settings className="w-4 h-4" />
        </button>
      </div>
    </header>
  )
}

function NavBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded text-sm transition-colors ${
        active ? 'bg-gray-800 text-emerald-400' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
      }`}
    >
      {children}
    </button>
  )
}
