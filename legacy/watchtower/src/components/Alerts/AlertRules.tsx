import { useState, useEffect } from 'react'
import { fetchAlertRules } from '../../api/client'
import type { AlertRule } from '../../types/sentinel'
import { Bell, BellOff, Shield } from 'lucide-react'

export function AlertRules() {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchAlertRules()
      .then(res => setRules(Array.isArray(res) ? res : res.rules || []))
      .catch(e => setError(e instanceof Error ? e.message : 'Failed to load'))
  }, [])

  return (
    <div className="p-4 space-y-4 overflow-y-auto h-full">
      <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Alert Rules</h2>

      {error && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300">{error}</div>
      )}

      {rules.length === 0 && !error && (
        <div className="text-center py-8">
          <Shield className="w-8 h-8 text-gray-600 mx-auto mb-2" />
          <p className="text-sm text-gray-500">No alert rules configured</p>
          <p className="text-xs text-gray-600 mt-1">Rules are managed on the SENTINEL backend</p>
        </div>
      )}

      {rules.map(rule => (
        <div key={rule.id} className="bg-gray-800/50 rounded-lg p-4 border border-gray-800">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              {rule.enabled ? (
                <Bell className="w-4 h-4 text-emerald-400" />
              ) : (
                <BellOff className="w-4 h-4 text-gray-600" />
              )}
              <span className="text-sm font-medium text-gray-200">{rule.name}</span>
            </div>
            <span className={`text-xs px-2 py-0.5 rounded ${rule.enabled ? 'bg-emerald-900/50 text-emerald-400' : 'bg-gray-800 text-gray-500'}`}>
              {rule.enabled ? 'Active' : 'Disabled'}
            </span>
          </div>

          {rule.description && (
            <p className="text-xs text-gray-400 mb-2">{rule.description}</p>
          )}

          <div className="space-y-1">
            {rule.conditions?.map((c, i) => (
              <div key={i} className="text-xs font-mono bg-gray-900/50 rounded px-2 py-1 text-gray-400">
                {c.field} {c.operator} {JSON.stringify(c.value)}
              </div>
            ))}
          </div>

          {rule.actions?.length > 0 && (
            <div className="mt-2 flex gap-1">
              {rule.actions.map((a, i) => (
                <span key={i} className="text-xs bg-gray-800 rounded px-2 py-0.5 text-gray-500">
                  {a.type}
                </span>
              ))}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
