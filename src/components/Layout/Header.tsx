import { useState, useEffect } from 'react'
import { Map, Activity, AlertTriangle, Settings, Radio, TrendingUp, BookOpen } from 'lucide-react'

export type View = 'map' | 'intel' | 'financial' | 'health' | 'alerts' | 'osint' | 'settings'

interface HeaderProps {
  currentView: View
  onViewChange: (v: View) => void
  isConnected: boolean
}

const TABS: { key: View; label: string; icon: React.ReactNode }[] = [
  { key: 'map', label: 'MAP', icon: <Map size={11} /> },
  { key: 'intel', label: 'INTEL', icon: <Radio size={11} /> },
  { key: 'financial', label: 'FINANCIAL', icon: <TrendingUp size={11} /> },
  { key: 'health', label: 'HEALTH', icon: <Activity size={11} /> },
  { key: 'alerts', label: 'ALERTS', icon: <AlertTriangle size={11} /> },
  { key: 'osint', label: 'OSINT', icon: <BookOpen size={11} /> },
  { key: 'settings', label: 'SETTINGS', icon: <Settings size={11} /> },
]

function useUTCClock() {
  const [time, setTime] = useState(() => new Date().toISOString().slice(11, 19) + 'Z')
  useEffect(() => {
    const iv = setInterval(() => {
      setTime(new Date().toISOString().slice(11, 19) + 'Z')
    }, 1000)
    return () => clearInterval(iv)
  }, [])
  return time
}

export function Header({ currentView, onViewChange, isConnected }: HeaderProps) {
  const utc = useUTCClock()

  return (
    <header className="h-9 min-h-9 max-h-9 bg-gray-950 border-b border-gray-800 flex items-center px-4 justify-between shrink-0 select-none">
      {/* Left: Branding */}
      <div className="flex items-center gap-2">
        <span className="text-[11px] font-mono uppercase tracking-wider text-cyan-500 font-bold">
          &#9670; SENTINEL V4
        </span>
      </div>

      {/* Center: View tabs */}
      <nav className="flex items-center gap-0.5">
        {TABS.map((tab) => (
          <button
            key={tab.key}
            onClick={() => onViewChange(tab.key)}
            className={`flex items-center gap-1 px-2.5 py-1 rounded text-[10px] font-mono uppercase tracking-wider transition-colors ${
              currentView === tab.key
                ? 'bg-cyan-950/60 text-cyan-400 border border-cyan-800/50'
                : 'text-gray-500 hover:text-gray-300 hover:bg-gray-900 border border-transparent'
            }`}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </nav>

      {/* Right: Connection + UTC */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span
            className={`w-1.5 h-1.5 rounded-full ${
              isConnected ? 'bg-emerald-400 shadow-[0_0_4px_rgba(52,211,153,0.6)]' : 'bg-red-500 shadow-[0_0_4px_rgba(239,68,68,0.6)]'
            }`}
          />
          <span className="text-[10px] font-mono uppercase tracking-wider text-gray-600">
            {isConnected ? 'ONLINE' : 'OFFLINE'}
          </span>
        </div>
        <span className="text-[10px] font-mono text-gray-500 tabular-nums">{utc}</span>
      </div>
    </header>
  )
}
