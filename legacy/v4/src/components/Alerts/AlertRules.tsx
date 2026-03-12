import { useState, useEffect, useCallback } from 'react'
import { fetchAlertRules, createAlertRule, updateAlertRule, deleteAlertRule } from '../../api/client'
import type { AlertRule } from '../../types/sentinel'
import { Bell, BellOff, Shield, Plus, Pencil, Trash2, X, Check, Loader2 } from 'lucide-react'

interface RuleFormData {
  name: string
  description: string
  enabled: boolean
  conditions: { field: string; operator: string; value: string }[]
  actions: { type: string; config: Record<string, string> }[]
}

const EMPTY_FORM: RuleFormData = {
  name: '',
  description: '',
  enabled: true,
  conditions: [{ field: 'category', operator: 'equals', value: '' }],
  actions: [{ type: 'notification', config: { min_severity: 'medium' } }],
}

const CONDITION_FIELDS = ['category', 'severity', 'magnitude', 'source'] as const
const CONDITION_OPERATORS: Record<string, string[]> = {
  category: ['equals', 'contains'],
  severity: ['equals', 'gt', 'lt'],
  magnitude: ['equals', 'gt', 'lt'],
  source: ['equals', 'contains'],
}
const ACTION_TYPES = ['notification', 'webhook', 'email', 'log'] as const
const SEVERITY_OPTIONS = ['low', 'medium', 'high', 'critical'] as const

function operatorLabel(op: string): string {
  switch (op) {
    case 'equals': return '='
    case 'contains': return 'contains'
    case 'gt': return '>'
    case 'lt': return '<'
    default: return op
  }
}

export function AlertRules() {
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [form, setForm] = useState<RuleFormData>({ ...EMPTY_FORM })
  const [saving, setSaving] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [toggling, setToggling] = useState<string | null>(null)

  const loadRules = useCallback(async () => {
    try {
      const res = await fetchAlertRules()
      setRules(Array.isArray(res) ? res : (res as any).rules || [])
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { loadRules() }, [loadRules])

  function openCreate() {
    setEditingId(null)
    setForm({ ...EMPTY_FORM, conditions: [{ field: 'category', operator: 'equals', value: '' }], actions: [{ type: 'notification', config: { min_severity: 'medium' } }] })
    setModalOpen(true)
  }

  function openEdit(rule: AlertRule) {
    setEditingId(rule.id)
    setForm({
      name: rule.name,
      description: rule.description || '',
      enabled: rule.enabled,
      conditions: (rule.conditions || []).map(c => ({ field: c.field, operator: c.operator, value: String(c.value ?? '') })),
      actions: (rule.actions || []).map(a => ({ type: a.type, config: { ...a.config } })),
    })
    if (form.conditions.length === 0) form.conditions = [{ field: 'category', operator: 'equals', value: '' }]
    if (form.actions.length === 0) form.actions = [{ type: 'notification', config: { min_severity: 'medium' } }]
    setModalOpen(true)
  }

  async function handleSave() {
    if (!form.name.trim()) return
    setSaving(true)
    try {
      const payload = {
        name: form.name.trim(),
        description: form.description.trim(),
        enabled: form.enabled,
        conditions: form.conditions.filter(c => c.value.trim()).map(c => ({
          field: c.field,
          operator: c.operator,
          value: c.field === 'magnitude' ? Number(c.value) : c.value,
        })),
        actions: form.actions.map(a => ({ type: a.type, config: a.config })),
      }
      if (editingId) {
        await updateAlertRule(editingId, payload)
      } else {
        await createAlertRule(payload)
      }
      setModalOpen(false)
      await loadRules()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteAlertRule(id)
      setDeleteConfirm(null)
      await loadRules()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed')
    }
  }

  async function handleToggle(rule: AlertRule) {
    setToggling(rule.id)
    try {
      await updateAlertRule(rule.id, { enabled: !rule.enabled })
      await loadRules()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Toggle failed')
    } finally {
      setToggling(null)
    }
  }

  function updateCondition(idx: number, patch: Partial<typeof form.conditions[0]>) {
    setForm(prev => {
      const conditions = [...prev.conditions]
      conditions[idx] = { ...conditions[idx], ...patch }
      if (patch.field && patch.field !== conditions[idx].field) {
        conditions[idx].operator = CONDITION_OPERATORS[patch.field]?.[0] || 'equals'
      }
      return { ...prev, conditions }
    })
  }

  function addCondition() {
    setForm(prev => ({ ...prev, conditions: [...prev.conditions, { field: 'category', operator: 'equals', value: '' }] }))
  }

  function removeCondition(idx: number) {
    setForm(prev => ({ ...prev, conditions: prev.conditions.filter((_, i) => i !== idx) }))
  }

  function updateAction(idx: number, patch: Partial<typeof form.actions[0]>) {
    setForm(prev => {
      const actions = [...prev.actions]
      actions[idx] = { ...actions[idx], ...patch }
      return { ...prev, actions }
    })
  }

  function addAction() {
    setForm(prev => ({ ...prev, actions: [...prev.actions, { type: 'notification', config: { min_severity: 'medium' } }] }))
  }

  function removeAction(idx: number) {
    setForm(prev => ({ ...prev, actions: prev.actions.filter((_, i) => i !== idx) }))
  }

  return (
    <div className="p-4 space-y-4 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Alert Rules</h2>
        <button
          onClick={openCreate}
          className="flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white transition-colors"
        >
          <Plus className="w-3.5 h-3.5" />
          Create Rule
        </button>
      </div>

      {error && (
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-400 hover:text-red-300"><X className="w-4 h-4" /></button>
        </div>
      )}

      {loading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 text-gray-500 animate-spin" />
        </div>
      )}

      {!loading && rules.length === 0 && !error && (
        <div className="text-center py-8">
          <Shield className="w-8 h-8 text-gray-600 mx-auto mb-2" />
          <p className="text-sm text-gray-500">No alert rules configured</p>
          <p className="text-xs text-gray-600 mt-1">Create a rule to get started</p>
        </div>
      )}

      {rules.map(rule => (
        <div key={rule.id} className="bg-gray-800/50 rounded-lg p-4 border border-gray-800">
          {/* Header */}
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              {rule.enabled ? (
                <Bell className="w-4 h-4 text-emerald-400" />
              ) : (
                <BellOff className="w-4 h-4 text-gray-600" />
              )}
              <span className="text-sm font-medium text-gray-200">{rule.name}</span>
            </div>
            <div className="flex items-center gap-2">
              {/* Toggle */}
              <button
                onClick={() => handleToggle(rule)}
                disabled={toggling === rule.id}
                className={`relative w-9 h-5 rounded-full transition-colors ${rule.enabled ? 'bg-emerald-600' : 'bg-gray-700'} ${toggling === rule.id ? 'opacity-50' : ''}`}
              >
                <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${rule.enabled ? 'translate-x-4' : 'translate-x-0'}`} />
              </button>
              {/* Edit */}
              <button
                onClick={() => openEdit(rule)}
                className="p-1.5 rounded hover:bg-gray-700 text-gray-500 hover:text-gray-300 transition-colors"
                title="Edit rule"
              >
                <Pencil className="w-3.5 h-3.5" />
              </button>
              {/* Delete */}
              {deleteConfirm === rule.id ? (
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => handleDelete(rule.id)}
                    className="p-1.5 rounded bg-red-700 hover:bg-red-600 text-white transition-colors"
                    title="Confirm delete"
                  >
                    <Check className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => setDeleteConfirm(null)}
                    className="p-1.5 rounded hover:bg-gray-700 text-gray-500 hover:text-gray-300 transition-colors"
                    title="Cancel"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => setDeleteConfirm(rule.id)}
                  className="p-1.5 rounded hover:bg-gray-700 text-gray-500 hover:text-red-400 transition-colors"
                  title="Delete rule"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>

          {rule.description && (
            <p className="text-xs text-gray-400 mb-2">{rule.description}</p>
          )}

          {/* Conditions summary */}
          {rule.conditions?.length > 0 && (
            <div className="space-y-1 mb-2">
              <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider">Conditions</p>
              {rule.conditions.map((c, i) => (
                <div key={i} className="text-xs font-mono bg-gray-900/50 rounded px-2 py-1 text-gray-400 flex items-center gap-1.5">
                  <span className="text-gray-300">{c.field}</span>
                  <span className="text-yellow-400">{operatorLabel(c.operator)}</span>
                  <span className="text-emerald-400">{JSON.stringify(c.value)}</span>
                </div>
              ))}
            </div>
          )}

          {/* Actions summary */}
          {rule.actions?.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold text-gray-500 uppercase tracking-wider mb-1">Actions</p>
              <div className="flex flex-wrap gap-1">
                {rule.actions.map((a, i) => (
                  <span key={i} className="text-xs bg-gray-800 rounded px-2 py-0.5 text-gray-400">
                    {a.type}
                    {a.config?.min_severity && <span className="text-gray-600 ml-1">({a.config.min_severity}+)</span>}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}

      {/* ── Modal ── */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setModalOpen(false)}>
          <div
            className="bg-gray-900 border border-gray-800 rounded-xl shadow-2xl w-full max-w-lg max-h-[85vh] overflow-y-auto mx-4"
            onClick={e => e.stopPropagation()}
          >
            {/* Modal header */}
            <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
              <h3 className="text-sm font-semibold text-gray-200">{editingId ? 'Edit Rule' : 'Create Rule'}</h3>
              <button onClick={() => setModalOpen(false)} className="text-gray-500 hover:text-gray-300">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="px-5 py-4 space-y-4">
              {/* Name */}
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Name</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="e.g. High severity earthquakes"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:outline-none focus:border-emerald-500"
                />
              </div>

              {/* Description */}
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Description</label>
                <input
                  type="text"
                  value={form.description}
                  onChange={e => setForm(prev => ({ ...prev, description: e.target.value }))}
                  placeholder="Optional description"
                  className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:outline-none focus:border-emerald-500"
                />
              </div>

              {/* Enabled toggle */}
              <div className="flex items-center justify-between">
                <label className="text-xs font-medium text-gray-400">Enabled</label>
                <button
                  type="button"
                  onClick={() => setForm(prev => ({ ...prev, enabled: !prev.enabled }))}
                  className={`relative w-9 h-5 rounded-full transition-colors ${form.enabled ? 'bg-emerald-600' : 'bg-gray-700'}`}
                >
                  <span className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${form.enabled ? 'translate-x-4' : 'translate-x-0'}`} />
                </button>
              </div>

              {/* Conditions */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Conditions</label>
                  <button onClick={addCondition} className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium">+ Add</button>
                </div>
                <div className="space-y-2">
                  {form.conditions.map((cond, idx) => (
                    <div key={idx} className="flex items-center gap-2 bg-gray-800/50 rounded-lg p-2">
                      {/* Field */}
                      <select
                        value={cond.field}
                        onChange={e => updateCondition(idx, { field: e.target.value })}
                        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-emerald-500"
                      >
                        {CONDITION_FIELDS.map(f => <option key={f} value={f}>{f}</option>)}
                      </select>
                      {/* Operator */}
                      <select
                        value={cond.operator}
                        onChange={e => updateCondition(idx, { operator: e.target.value })}
                        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-emerald-500"
                      >
                        {(CONDITION_OPERATORS[cond.field] || ['equals']).map(op => (
                          <option key={op} value={op}>{operatorLabel(op)}</option>
                        ))}
                      </select>
                      {/* Value */}
                      {cond.field === 'severity' ? (
                        <select
                          value={cond.value}
                          onChange={e => updateCondition(idx, { value: e.target.value })}
                          className="flex-1 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-emerald-500"
                        >
                          <option value="">Select...</option>
                          {SEVERITY_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
                        </select>
                      ) : (
                        <input
                          type={cond.field === 'magnitude' ? 'number' : 'text'}
                          value={cond.value}
                          onChange={e => updateCondition(idx, { value: e.target.value })}
                          placeholder="Value"
                          className="flex-1 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 placeholder-gray-600 focus:outline-none focus:border-emerald-500"
                        />
                      )}
                      {form.conditions.length > 1 && (
                        <button onClick={() => removeCondition(idx)} className="text-gray-600 hover:text-red-400 transition-colors">
                          <X className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Actions */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Actions</label>
                  <button onClick={addAction} className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium">+ Add</button>
                </div>
                <div className="space-y-2">
                  {form.actions.map((action, idx) => (
                    <div key={idx} className="flex items-center gap-2 bg-gray-800/50 rounded-lg p-2">
                      {/* Type */}
                      <select
                        value={action.type}
                        onChange={e => updateAction(idx, { type: e.target.value })}
                        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-emerald-500"
                      >
                        {ACTION_TYPES.map(t => <option key={t} value={t}>{t}</option>)}
                      </select>
                      {/* Severity threshold */}
                      <select
                        value={action.config.min_severity || 'medium'}
                        onChange={e => updateAction(idx, { config: { ...action.config, min_severity: e.target.value } })}
                        className="flex-1 bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-gray-200 focus:outline-none focus:border-emerald-500"
                      >
                        {SEVERITY_OPTIONS.map(s => <option key={s} value={s}>{s}+</option>)}
                      </select>
                      {form.actions.length > 1 && (
                        <button onClick={() => removeAction(idx)} className="text-gray-600 hover:text-red-400 transition-colors">
                          <X className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Modal footer */}
            <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-gray-800">
              <button
                onClick={() => setModalOpen(false)}
                className="px-4 py-2 text-xs font-medium text-gray-400 hover:text-gray-200 rounded-lg hover:bg-gray-800 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={saving || !form.name.trim()}
                className="flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                {editingId ? 'Update Rule' : 'Create Rule'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
