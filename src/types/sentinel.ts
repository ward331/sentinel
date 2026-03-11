export interface Location {
  type: 'Point' | 'Polygon'
  coordinates: number[] | number[][]
  bbox?: number[]
}

export interface Badge {
  label: string
  type: 'source' | 'precision' | 'freshness' | 'filter'
  timestamp: string
}

export interface SentinelEvent {
  id: string
  title: string
  description: string
  source: string
  source_id: string
  occurred_at: string
  ingested_at: string
  location: Location
  precision: 'exact' | 'polygon_area' | 'approximate' | 'text_inferred' | 'unknown'
  magnitude: number
  category: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  metadata: Record<string, string>
  badges: Badge[]
}

export interface EventListResponse {
  events: SentinelEvent[]
  total: number
  limit: number
  offset: number
}

export interface ProviderInfo {
  name: string
  interval_seconds: number
  enabled: boolean
}

export interface ProviderListResponse {
  providers: ProviderInfo[]
  total: number
}

export interface HealthCheck {
  name: string
  status: 'healthy' | 'degraded' | 'unhealthy'
  message: string
  details: Record<string, unknown>
  duration_ms: number
}

export interface HealthResponse {
  status: 'healthy' | 'degraded' | 'unhealthy'
  timestamp: string
  uptime_seconds: number
  checks?: Record<string, HealthCheck>
}

export interface MetricsResponse {
  events_processed_total: number
  events_by_provider: Record<string, number>
  uptime_seconds: number
  start_time: string
}

export interface AlertCondition {
  field: string
  operator: string
  value: unknown
}

export interface AlertAction {
  type: string
  config: Record<string, string>
}

export interface AlertRule {
  id: string
  name: string
  description: string
  enabled: boolean
  conditions: AlertCondition[]
  actions: AlertAction[]
  created_at: string
  updated_at: string
}

export interface EventFilters {
  source?: string
  category?: string
  severity?: string
  exclude_category?: string
  exclude_source?: string
  min_magnitude?: number
  max_magnitude?: number
  q?: string
  start_time?: string
  end_time?: string
  bbox?: string
  limit?: number
  offset?: number
}

export interface WatchtowerConfig {
  serverUrl: string
  configured: boolean
}
