import { useState, useCallback, useMemo } from 'react'
import { getConfig, clearConfig } from './api/client'
import { useEvents } from './hooks/useEvents'
import { SetupWizard } from './components/Setup/SetupWizard'
import { Header, type View } from './components/Layout/Header'
import { EventMap } from './components/Map/EventMap'
import { EventFeed } from './components/Feed/EventFeed'
import { FilterPanel } from './components/Filters/FilterPanel'
import { ProviderHealth } from './components/Health/ProviderHealth'
import { AlertRules } from './components/Alerts/AlertRules'
import { EventDetail } from './components/Layout/EventDetail'
import type { SentinelEvent, EventFilters } from './types/sentinel'

function App() {
  const [configured, setConfigured] = useState(getConfig().configured)
  const [view, setView] = useState<View>('map')
  const [filters, setFilters] = useState<EventFilters>({})
  const [selectedEvent, setSelectedEvent] = useState<SentinelEvent | null>(null)
  const { events, loading } = useEvents(configured ? filters : {})

  const sources = useMemo(() => {
    const s = new Set(events.map(e => e.source).filter(Boolean))
    return Array.from(s).sort()
  }, [events])

  const categories = useMemo(() => {
    const c = new Set(events.map(e => e.category).filter(Boolean))
    return Array.from(c).sort()
  }, [events])

  const handleSetupComplete = useCallback(() => setConfigured(true), [])
  const handleOpenSettings = useCallback(() => {
    if (confirm('Disconnect from SENTINEL server and reconfigure?')) {
      clearConfig()
      setConfigured(false)
    }
  }, [])

  if (!configured) {
    return <SetupWizard onComplete={handleSetupComplete} />
  }

  return (
    <div className="h-screen flex flex-col">
      <Header
        view={view}
        onViewChange={setView}
        onOpenSettings={handleOpenSettings}
        connected={configured}
        eventCount={events.length}
      />

      <div className="flex-1 flex overflow-hidden">
        {view === 'map' && (
          <>
            {/* Sidebar: filters + feed */}
            <div className="w-80 border-r border-gray-800 flex flex-col shrink-0 bg-gray-900">
              <FilterPanel
                filters={filters}
                onChange={setFilters}
                sources={sources}
                categories={categories}
              />
              <div className="flex-1 overflow-hidden">
                <EventFeed
                  events={events}
                  onSelectEvent={setSelectedEvent}
                  loading={loading}
                />
              </div>
            </div>

            {/* Map area */}
            <div className="flex-1 relative">
              <EventMap events={events} onSelectEvent={setSelectedEvent} />
              {selectedEvent && (
                <EventDetail
                  event={selectedEvent}
                  onClose={() => setSelectedEvent(null)}
                />
              )}
            </div>
          </>
        )}

        {view === 'health' && (
          <div className="flex-1">
            <ProviderHealth />
          </div>
        )}

        {view === 'alerts' && (
          <div className="flex-1">
            <AlertRules />
          </div>
        )}
      </div>
    </div>
  )
}

export default App
