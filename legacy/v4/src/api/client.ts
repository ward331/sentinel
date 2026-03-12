import type {
  EventListResponse, EventFilters, HealthResponse, MetricsResponse,
  ProviderListResponse, AlertRule, WatchtowerConfig, SignalBoard,
  CorrelationListResponse, NewsResponse, IntelBriefing, FinancialOverview,
  EntitySearchResponse, OSINTResourceListResponse, OSINTResource,
  NotificationConfig, ProximityConfig, ProximityEventsResponse, UIConfig,
} from '../types/sentinel'

const CONFIG_KEY = 'watchtower_config'

export const DEFAULT_SERVER_URL = 'http://127.0.0.1:8080'

export function getConfig(): WatchtowerConfig {
  const raw = localStorage.getItem(CONFIG_KEY)
  if (!raw) return { serverUrl: '', configured: true }
  try { return JSON.parse(raw) } catch { return { serverUrl: '', configured: true } }
}

export function saveConfig(serverUrl: string) {
  const url = serverUrl.replace(/\/+$/, '')
  localStorage.setItem(CONFIG_KEY, JSON.stringify({ serverUrl: url, configured: true }))
}

export function clearConfig() {
  localStorage.removeItem(CONFIG_KEY)
}

function baseUrl(): string {
  return getConfig().serverUrl || ''
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

// ─── Connection ──────────────────────────────────────────────────

export async function testConnection(serverUrl: string): Promise<HealthResponse> {
  const url = serverUrl.replace(/\/+$/, '') || ''
  const res = await fetch(`${url}/api/health?detailed=true`, { signal: AbortSignal.timeout(5000) })
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json()
}

export function sseUrl(): string {
  return `${baseUrl()}/api/events/stream`
}

export function wsUrl(): string {
  const base = baseUrl() || window.location.origin
  return base.replace(/^http/, 'ws') + '/api/ws'
}

// ─── Events ──────────────────────────────────────────────────────

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

export async function acknowledgeEvent(id: string) {
  return api(`/api/events/${id}/acknowledge`, { method: 'POST' })
}

// ─── Health & Providers ──────────────────────────────────────────

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

export async function fetchProviderStats(name: string) {
  return api<Record<string, unknown>>(`/api/providers/${name}/stats`)
}

// ─── Alert Rules ─────────────────────────────────────────────────

export async function fetchAlertRules(): Promise<AlertRule[]> {
  return api('/api/alerts/rules')
}

export async function createAlertRule(rule: Partial<AlertRule>): Promise<AlertRule> {
  return api('/api/alerts/rules', { method: 'POST', body: JSON.stringify(rule) })
}

export async function updateAlertRule(id: string, rule: Partial<AlertRule>) {
  return api(`/api/alerts/rules/${id}`, { method: 'PUT', body: JSON.stringify(rule) })
}

export async function deleteAlertRule(id: string) {
  return api(`/api/alerts/rules/${id}`, { method: 'DELETE' })
}

// ─── Signal Board ────────────────────────────────────────────────

export async function fetchSignalBoard(): Promise<SignalBoard> {
  return api('/api/signal-board')
}

// ─── Correlations ────────────────────────────────────────────────

export async function fetchCorrelations(): Promise<CorrelationListResponse> {
  return api('/api/correlations')
}

// ─── News ────────────────────────────────────────────────────────

export async function fetchNews(limit = 50): Promise<NewsResponse> {
  return api(`/api/news?limit=${limit}`)
}

// ─── Intel Briefing ──────────────────────────────────────────────

export async function fetchIntelBriefing(): Promise<IntelBriefing> {
  return api('/api/intel/briefing')
}

// ─── Financial ───────────────────────────────────────────────────

export async function fetchFinancialOverview(): Promise<FinancialOverview> {
  return api('/api/financial/overview')
}

// ─── Entity Search ───────────────────────────────────────────────

export async function searchEntities(query: string): Promise<EntitySearchResponse> {
  return api(`/api/entity/search?q=${encodeURIComponent(query)}`)
}

// ─── OSINT Resources ─────────────────────────────────────────────

export async function fetchOsintResources(params: Record<string, string> = {}): Promise<OSINTResourceListResponse> {
  const qs = new URLSearchParams(params)
  if (!qs.has('limit')) qs.set('limit', '100')
  return api(`/api/osint/resources?${qs}`)
}

export async function fetchOsintCategories(): Promise<string[]> {
  return api('/api/osint/resources/categories')
}

export async function fetchOsintPlatforms(): Promise<string[]> {
  return api('/api/osint/resources/platforms')
}

export async function fetchContextualResources(eventId: string) {
  return api<{ event_id: string; resources: OSINTResource[] }>(`/api/osint/resources/contextual/${eventId}`)
}

// ─── Notifications ───────────────────────────────────────────────

export async function fetchNotificationConfig(): Promise<NotificationConfig> {
  return api('/api/notifications/config')
}

export async function updateNotificationConfig(config: Partial<NotificationConfig>) {
  return api('/api/notifications/config', { method: 'POST', body: JSON.stringify(config) })
}

export async function testNotificationChannel(channel: string) {
  return api<{ channel: string; status: string; message: string }>(`/api/notifications/test/${channel}`, { method: 'POST' })
}

// ─── Proximity ───────────────────────────────────────────────────

export async function fetchProximityConfig(): Promise<ProximityConfig> {
  return api('/api/proximity/config')
}

export async function updateProximityConfig(config: { lat: number; lon: number; radius_km: number }) {
  return api('/api/proximity/config', { method: 'POST', body: JSON.stringify(config) })
}

export async function fetchProximityEvents(minutes = 60, severity?: string): Promise<ProximityEventsResponse> {
  const params = new URLSearchParams({ minutes: String(minutes) })
  if (severity) params.set('severity', severity)
  return api(`/api/proximity/events?${params}`)
}

// ─── Config ──────────────────────────────────────────────────────

export async function fetchServerConfig(): Promise<Record<string, unknown>> {
  return api('/api/config')
}

export async function updateServerConfig(updates: Record<string, unknown>) {
  return api('/api/config', { method: 'POST', body: JSON.stringify(updates) })
}

export async function fetchUIConfig(): Promise<UIConfig> {
  return api('/api/config/ui')
}
