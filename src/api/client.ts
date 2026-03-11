import type { EventListResponse, EventFilters, HealthResponse, MetricsResponse, ProviderListResponse, AlertRule, WatchtowerConfig } from '../types/sentinel'

const CONFIG_KEY = 'watchtower_config'

export function getConfig(): WatchtowerConfig {
  const raw = localStorage.getItem(CONFIG_KEY)
  if (!raw) return { serverUrl: '', configured: false }
  try {
    return JSON.parse(raw)
  } catch {
    return { serverUrl: '', configured: false }
  }
}

export function saveConfig(serverUrl: string) {
  const url = serverUrl.replace(/\/+$/, '')
  localStorage.setItem(CONFIG_KEY, JSON.stringify({ serverUrl: url, configured: true }))
}

export function clearConfig() {
  localStorage.removeItem(CONFIG_KEY)
}

function baseUrl(): string {
  return getConfig().serverUrl
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${baseUrl()}${path}`
  const res = await fetch(url, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new Error(`${res.status} ${res.statusText}: ${text}`)
  }
  return res.json()
}

export async function testConnection(serverUrl: string): Promise<HealthResponse> {
  const url = serverUrl.replace(/\/+$/, '')
  const res = await fetch(`${url}/api/health?detailed=true`, { signal: AbortSignal.timeout(5000) })
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json()
}

export async function fetchEvents(filters: EventFilters = {}): Promise<EventListResponse> {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) {
    if (v !== undefined && v !== '' && v !== null) params.set(k, String(v))
  }
  if (!params.has('limit')) params.set('limit', '200')
  return api(`/api/events?${params}`)
}

export async function fetchEvent(id: string) {
  return api(`/api/events/${id}`)
}

export async function fetchHealth(): Promise<HealthResponse> {
  return api('/api/health?detailed=true')
}

export async function fetchMetrics(): Promise<MetricsResponse> {
  return api('/api/metrics')
}

export async function fetchProviders(): Promise<ProviderListResponse> {
  return api('/api/providers')
}

export async function fetchProviderHealth() {
  return api<Record<string, unknown>>('/api/providers/health')
}

export async function fetchAlertRules(): Promise<AlertRule[]> {
  return api('/api/alerts/rules')
}

export async function createAlertRule(rule: Partial<AlertRule>): Promise<AlertRule> {
  return api('/api/alerts/rules', { method: 'POST', body: JSON.stringify(rule) })
}

export async function fetchServerConfig(): Promise<any> {
  return api('/api/config')
}

export async function updateServerConfig(updates: Record<string, any>): Promise<any> {
  return api('/api/config', { method: 'POST', body: JSON.stringify(updates) })
}

export function sseUrl(): string {
  return `${baseUrl()}/api/events/stream`
}
