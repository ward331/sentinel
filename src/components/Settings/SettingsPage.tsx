import { useState, useEffect } from 'react'
import { Eye, EyeOff, Save, Unplug, RefreshCw, ChevronDown, ChevronRight, BookOpen } from 'lucide-react'
import { fetchServerConfig, updateServerConfig, getConfig } from '../../api/client'
import { ApiCatalog } from './ApiCatalog'

// ─── Types ───────────────────────────────────────────────────────────
interface ApiKeyEntry {
  key: string
  label: string
  category: string
  url: string
}

const API_KEYS: ApiKeyEntry[] = [
  { key: 'adsbexchange', label: 'ADS-B Exchange', category: 'Aviation', url: 'https://www.adsbexchange.com/data/' },
  { key: 'acled', label: 'ACLED', category: 'Conflict', url: 'https://developer.acleddata.com/' },
  { key: 'nasa', label: 'NASA FIRMS', category: 'Environmental', url: 'https://firms.modaps.eosdis.nasa.gov/api/area/' },
  { key: 'spacetrack', label: 'SpaceTrack', category: 'Space', url: 'https://www.space-track.org/auth/createAccount' },
  { key: 'n2yo', label: 'N2YO', category: 'Space', url: 'https://www.n2yo.com/api/' },
  { key: 'alpha_vantage', label: 'Alpha Vantage', category: 'Financial', url: 'https://www.alphavantage.co/support/#api-key' },
]

interface ProviderConfig {
  name: string
  enabled: boolean
  interval_seconds: number
  category?: string
  config?: Record<string, unknown>
}

interface Props {
  onDisconnect: () => void
}

// ─── Component ───────────────────────────────────────────────────────
type SettingsTab = 'config' | 'catalog'

export function SettingsPage({ onDisconnect }: Props) {
  const [tab, setTab] = useState<SettingsTab>('config')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // API Keys state
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
  const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set())

  // Provider state
  const [providers, setProviders] = useState<ProviderConfig[]>([])
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())

  // General state
  const [retentionDays, setRetentionDays] = useState(30)

  const serverUrl = getConfig().serverUrl

  // Load config on mount
  useEffect(() => {
    loadConfig()
  }, [])

  async function loadConfig() {
    setLoading(true)
    setError(null)
    try {
      const config = await fetchServerConfig()
      // Extract API keys (they come back as masked or empty)
      const keys: Record<string, string> = {}
      if (config.api_keys) {
        for (const [k, v] of Object.entries(config.api_keys)) {
          keys[k] = (v as string) || ''
        }
      }
      setApiKeys(keys)

      // Extract providers
      if (config.providers && Array.isArray(config.providers)) {
        setProviders(config.providers)
      }

      // Extract general settings
      if (config.retention_days !== undefined) {
        setRetentionDays(Number(config.retention_days) || 30)
      }
    } catch (err) {
      setError(`Failed to load config: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setLoading(false)
    }
  }

  function flash(msg: string) {
    setSuccess(msg)
    setTimeout(() => setSuccess(null), 3000)
  }

  async function saveApiKeys() {
    setSaving('keys')
    setError(null)
    try {
      await updateServerConfig({ api_keys: apiKeys })
      flash('API keys saved')
      await loadConfig()
    } catch (err) {
      setError(`Failed to save keys: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(null)
    }
  }

  async function saveProviders() {
    setSaving('providers')
    setError(null)
    try {
      await updateServerConfig({ providers })
      flash('Provider configuration saved')
      await loadConfig()
    } catch (err) {
      setError(`Failed to save providers: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(null)
    }
  }

  async function saveGeneral() {
    setSaving('general')
    setError(null)
    try {
      await updateServerConfig({ retention_days: retentionDays })
      flash('General settings saved')
    } catch (err) {
      setError(`Failed to save settings: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(null)
    }
  }

  function toggleReveal(key: string) {
    setRevealedKeys(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  function updateProvider(index: number, updates: Partial<ProviderConfig>) {
    setProviders(prev => prev.map((p, i) => i === index ? { ...p, ...updates } : p))
  }

  function updateProviderConfig(index: number, field: string, value: unknown) {
    setProviders(prev => prev.map((p, i) => {
      if (i !== index) return p
      return { ...p, config: { ...(p.config || {}), [field]: value } }
    }))
  }

  function toggleProviderGroup(group: string) {
    setCollapsedGroups(prev => {
      const next = new Set(prev)
      if (next.has(group)) next.delete(group)
      else next.add(group)
      return next
    })
  }

  // Group providers by category
  const providersByCategory: Record<string, { providers: ProviderConfig[]; indices: number[] }> = {}
  providers.forEach((p, i) => {
    const cat = p.category || 'Other'
    if (!providersByCategory[cat]) providersByCategory[cat] = { providers: [], indices: [] }
    providersByCategory[cat].providers.push(p)
    providersByCategory[cat].indices.push(i)
  })

  // Group API keys by category
  const keysByCategory: Record<string, ApiKeyEntry[]> = {}
  for (const entry of API_KEYS) {
    if (!keysByCategory[entry.category]) keysByCategory[entry.category] = []
    keysByCategory[entry.category].push(entry)
  }

  function hasStoredKey(key: string): boolean {
    const val = apiKeys[key]
    return !!val && val !== ''
  }

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-gray-950 text-gray-400">
        <RefreshCw className="w-5 h-5 animate-spin mr-2" />
        Loading configuration...
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto bg-gray-950">
      <div className="max-w-3xl mx-auto px-6 py-8 space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-white">Settings</h1>
          <div className="flex items-center bg-gray-900 rounded-lg border border-gray-800 p-0.5">
            <button
              onClick={() => setTab('config')}
              className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
                tab === 'config' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-gray-300'
              }`}
            >
              Configuration
            </button>
            <button
              onClick={() => setTab('catalog')}
              className={`flex items-center gap-1.5 px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
                tab === 'catalog' ? 'bg-gray-800 text-white' : 'text-gray-500 hover:text-gray-300'
              }`}
            >
              <BookOpen className="w-3.5 h-3.5" />
              API Catalog
            </button>
          </div>
        </div>

        {tab === 'catalog' && <ApiCatalog />}

        {tab === 'config' && <>
        {/* Status messages */}
        {error && (
          <div className="px-4 py-3 rounded bg-red-900/30 border border-red-800 text-red-300 text-sm">
            {error}
          </div>
        )}
        {success && (
          <div className="px-4 py-3 rounded bg-emerald-900/30 border border-emerald-800 text-emerald-300 text-sm">
            {success}
          </div>
        )}

        {/* ─── Connection ─────────────────────────────────────────── */}
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Connection</h2>
          <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-300">Server URL</div>
                <div className="text-xs text-gray-500 font-mono mt-1">{serverUrl}</div>
              </div>
              <button
                onClick={onDisconnect}
                className="flex items-center gap-2 px-3 py-1.5 rounded bg-red-900/30 border border-red-800 text-red-400 text-sm hover:bg-red-900/50 transition-colors"
              >
                <Unplug className="w-4 h-4" />
                Disconnect
              </button>
            </div>
          </div>
        </section>

        <div className="border-t border-gray-800" />

        {/* ─── API Keys ───────────────────────────────────────────── */}
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">API Keys</h2>
          <div className="bg-gray-900 rounded-lg border border-gray-800 divide-y divide-gray-800">
            {Object.entries(keysByCategory).map(([category, entries]) => (
              <div key={category} className="p-4 space-y-3">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider">{category}</div>
                {entries.map(entry => (
                  <div key={entry.key} className="flex items-center gap-3">
                    <span className={`w-2 h-2 rounded-full shrink-0 ${hasStoredKey(entry.key) ? 'bg-emerald-400' : 'bg-gray-600'}`} />
                    <a href={entry.url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-400 hover:text-blue-300 hover:underline w-32 shrink-0 cursor-pointer">{entry.label}</a>
                    <div className="flex-1 flex items-center gap-1">
                      <input
                        type={revealedKeys.has(entry.key) ? 'text' : 'password'}
                        value={apiKeys[entry.key] || ''}
                        onChange={e => setApiKeys(prev => ({ ...prev, [entry.key]: e.target.value }))}
                        placeholder="Not configured"
                        className="flex-1 bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm text-gray-300 placeholder-gray-600 font-mono outline-none focus:border-gray-600"
                      />
                      <button
                        onClick={() => toggleReveal(entry.key)}
                        className="p-1.5 rounded hover:bg-gray-800 transition-colors shrink-0"
                      >
                        {revealedKeys.has(entry.key)
                          ? <EyeOff className="w-4 h-4 text-gray-500" />
                          : <Eye className="w-4 h-4 text-gray-500" />
                        }
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            ))}
            <div className="p-4 flex justify-end">
              <button
                onClick={saveApiKeys}
                disabled={saving === 'keys'}
                className="flex items-center gap-2 px-4 py-2 rounded bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
              >
                <Save className="w-4 h-4" />
                {saving === 'keys' ? 'Saving...' : 'Save Keys'}
              </button>
            </div>
          </div>
        </section>

        <div className="border-t border-gray-800" />

        {/* ─── Provider Configuration ─────────────────────────────── */}
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Provider Configuration</h2>
          <div className="bg-gray-900 rounded-lg border border-gray-800 divide-y divide-gray-800">
            {Object.entries(providersByCategory).map(([category, { providers: catProviders, indices }]) => {
              const isCollapsed = collapsedGroups.has(category)
              return (
                <div key={category}>
                  <div
                    className="flex items-center gap-2 px-4 py-3 cursor-pointer select-none hover:bg-gray-800/50 transition-colors"
                    onClick={() => toggleProviderGroup(category)}
                  >
                    {isCollapsed
                      ? <ChevronRight className="w-4 h-4 text-gray-500" />
                      : <ChevronDown className="w-4 h-4 text-gray-500" />
                    }
                    <span className="text-sm font-medium text-gray-300">{category}</span>
                    <span className="text-xs text-gray-600">
                      {catProviders.filter(p => p.enabled).length}/{catProviders.length} enabled
                    </span>
                  </div>
                  {!isCollapsed && (
                    <div className="px-4 pb-4 space-y-3">
                      {catProviders.map((provider, ci) => {
                        const globalIndex = indices[ci]
                        const isAdsb = provider.name.toLowerCase().includes('adsb') || provider.name.toLowerCase().includes('ads-b')
                        const isOpenMeteo = provider.name.toLowerCase().includes('openmeteo') || provider.name.toLowerCase().includes('open_meteo')

                        return (
                          <div key={provider.name} className="rounded bg-gray-800/40 p-3 space-y-2">
                            <div className="flex items-center justify-between">
                              <div className="flex items-center gap-3">
                                <label className="relative inline-flex items-center cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={provider.enabled}
                                    onChange={e => updateProvider(globalIndex, { enabled: e.target.checked })}
                                    className="sr-only peer"
                                  />
                                  <div className="w-8 h-4 bg-gray-700 peer-checked:bg-emerald-600 rounded-full transition-colors after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-3 after:w-3 after:transition-all peer-checked:after:translate-x-4" />
                                </label>
                                <span className="text-sm text-gray-300 font-mono">{provider.name}</span>
                              </div>
                              <div className="flex items-center gap-2">
                                <label className="text-xs text-gray-500">Interval</label>
                                <input
                                  type="number"
                                  value={provider.interval_seconds}
                                  onChange={e => updateProvider(globalIndex, { interval_seconds: Number(e.target.value) })}
                                  min={10}
                                  className="w-20 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 text-right outline-none focus:border-gray-600"
                                />
                                <span className="text-xs text-gray-500">sec</span>
                              </div>
                            </div>

                            {/* ADS-B specific fields */}
                            {isAdsb && (
                              <div className="flex gap-2 pt-1">
                                <div className="flex items-center gap-1">
                                  <label className="text-xs text-gray-500">Lat</label>
                                  <input
                                    type="number"
                                    value={(provider.config?.lat as number) ?? ''}
                                    onChange={e => updateProviderConfig(globalIndex, 'lat', Number(e.target.value))}
                                    step="0.01"
                                    className="w-20 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 outline-none focus:border-gray-600"
                                    placeholder="0.0"
                                  />
                                </div>
                                <div className="flex items-center gap-1">
                                  <label className="text-xs text-gray-500">Lon</label>
                                  <input
                                    type="number"
                                    value={(provider.config?.lon as number) ?? ''}
                                    onChange={e => updateProviderConfig(globalIndex, 'lon', Number(e.target.value))}
                                    step="0.01"
                                    className="w-20 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 outline-none focus:border-gray-600"
                                    placeholder="0.0"
                                  />
                                </div>
                                <div className="flex items-center gap-1">
                                  <label className="text-xs text-gray-500">Radius (nm)</label>
                                  <input
                                    type="number"
                                    value={(provider.config?.radius as number) ?? ''}
                                    onChange={e => updateProviderConfig(globalIndex, 'radius', Number(e.target.value))}
                                    min={1}
                                    className="w-16 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 outline-none focus:border-gray-600"
                                    placeholder="250"
                                  />
                                </div>
                              </div>
                            )}

                            {/* OpenMeteo specific fields */}
                            {isOpenMeteo && (
                              <div className="flex gap-2 pt-1">
                                <div className="flex items-center gap-1">
                                  <label className="text-xs text-gray-500">Lat</label>
                                  <input
                                    type="number"
                                    value={(provider.config?.lat as number) ?? ''}
                                    onChange={e => updateProviderConfig(globalIndex, 'lat', Number(e.target.value))}
                                    step="0.01"
                                    className="w-20 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 outline-none focus:border-gray-600"
                                    placeholder="0.0"
                                  />
                                </div>
                                <div className="flex items-center gap-1">
                                  <label className="text-xs text-gray-500">Lon</label>
                                  <input
                                    type="number"
                                    value={(provider.config?.lon as number) ?? ''}
                                    onChange={e => updateProviderConfig(globalIndex, 'lon', Number(e.target.value))}
                                    step="0.01"
                                    className="w-20 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-300 outline-none focus:border-gray-600"
                                    placeholder="0.0"
                                  />
                                </div>
                              </div>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  )}
                </div>
              )
            })}
            {providers.length === 0 && (
              <div className="p-4 text-sm text-gray-500 text-center">
                No providers configured. Connect to a SENTINEL server to see providers.
              </div>
            )}
            <div className="p-4 flex justify-end">
              <button
                onClick={saveProviders}
                disabled={saving === 'providers'}
                className="flex items-center gap-2 px-4 py-2 rounded bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
              >
                <Save className="w-4 h-4" />
                {saving === 'providers' ? 'Saving...' : 'Save Providers'}
              </button>
            </div>
          </div>
        </section>

        <div className="border-t border-gray-800" />

        {/* ─── General ────────────────────────────────────────────── */}
        <section className="space-y-3">
          <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">General</h2>
          <div className="bg-gray-900 rounded-lg border border-gray-800 p-4 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-300">Data retention</div>
                <div className="text-xs text-gray-500 mt-0.5">Events older than this will be purged</div>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="number"
                  value={retentionDays}
                  onChange={e => setRetentionDays(Number(e.target.value))}
                  min={1}
                  max={365}
                  className="w-20 bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm text-gray-300 text-right outline-none focus:border-gray-600"
                />
                <span className="text-sm text-gray-500">days</span>
              </div>
            </div>
            <div className="flex justify-end pt-2 border-t border-gray-800">
              <button
                onClick={saveGeneral}
                disabled={saving === 'general'}
                className="flex items-center gap-2 px-4 py-2 rounded bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
              >
                <Save className="w-4 h-4" />
                {saving === 'general' ? 'Saving...' : 'Save Settings'}
              </button>
            </div>
          </div>
        </section>

        <div className="h-8" />
        </>}
      </div>
    </div>
  )
}
