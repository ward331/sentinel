import { useState } from 'react'
import { testConnection, saveConfig } from '../../api/client'
import type { HealthResponse } from '../../types/sentinel'
import { Shield, CheckCircle, XCircle, Loader2 } from 'lucide-react'

interface Props {
  onComplete: () => void
}

export function SetupWizard({ onComplete }: Props) {
  const [url, setUrl] = useState('http://localhost:8080')
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [health, setHealth] = useState<HealthResponse | null>(null)

  async function handleTest() {
    setTesting(true)
    setError(null)
    setHealth(null)
    try {
      const h = await testConnection(url)
      setHealth(h)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Connection failed')
    } finally {
      setTesting(false)
    }
  }

  function handleConnect() {
    saveConfig(url)
    onComplete()
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-950 p-4">
      <div className="max-w-md w-full bg-gray-900 rounded-xl border border-gray-800 p-8">
        <div className="flex items-center gap-3 mb-6">
          <Shield className="w-10 h-10 text-emerald-400" />
          <div>
            <h1 className="text-2xl font-bold text-white">Watchtower</h1>
            <p className="text-sm text-gray-400">SENTINEL V2 Frontend</p>
          </div>
        </div>

        <p className="text-gray-300 mb-6">
          Enter the URL of your SENTINEL V2 server to get started.
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Server URL</label>
            <input
              type="url"
              value={url}
              onChange={e => setUrl(e.target.value)}
              placeholder="http://localhost:8080"
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-emerald-500"
              onKeyDown={e => e.key === 'Enter' && handleTest()}
            />
          </div>

          <button
            onClick={handleTest}
            disabled={testing || !url}
            className="w-full bg-gray-800 hover:bg-gray-700 border border-gray-600 text-white py-2.5 rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : null}
            {testing ? 'Testing...' : 'Test Connection'}
          </button>

          {error && (
            <div className="flex items-start gap-2 bg-red-900/30 border border-red-800 rounded-lg p-3">
              <XCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
              <p className="text-sm text-red-300">{error}</p>
            </div>
          )}

          {health && (
            <div className="bg-emerald-900/20 border border-emerald-800 rounded-lg p-4 space-y-2">
              <div className="flex items-center gap-2">
                <CheckCircle className="w-5 h-5 text-emerald-400" />
                <span className="font-medium text-emerald-300">Connected!</span>
              </div>
              <div className="text-sm text-gray-300 space-y-1">
                <p>Status: <span className="text-emerald-400">{health.status}</span></p>
                <p>Uptime: {Math.floor(health.uptime_seconds / 3600)}h {Math.floor((health.uptime_seconds % 3600) / 60)}m</p>
              </div>

              <button
                onClick={handleConnect}
                className="w-full mt-3 bg-emerald-600 hover:bg-emerald-500 text-white py-2.5 rounded-lg font-medium transition-colors"
              >
                Launch Watchtower
              </button>
            </div>
          )}
        </div>

        <p className="text-xs text-gray-600 mt-6 text-center">
          Watchtower connects to any SENTINEL V2 instance via REST + SSE
        </p>
      </div>
    </div>
  )
}
