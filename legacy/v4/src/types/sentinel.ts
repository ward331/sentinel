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
  truth_score: number
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
  display_name: string
  category: string
  tier: string
  interval_seconds: number
  enabled: boolean
  status: string
  events_last_hour: number
  key_file?: string
  signup_url?: string
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
  version?: string
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
  truth_score_min?: number
  country?: string
  limit?: number
  offset?: number
}

export interface WatchtowerConfig {
  serverUrl: string
  configured: boolean
}

// ─── V3 Types ────────────────────────────────────────────────────

export interface SignalBoard {
  military: number
  cyber: number
  financial: number
  natural: number
  health: number
  calculated_at: string
  active_alerts?: number
  active_correlations?: number
}

export interface CorrelationFlash {
  id: number
  region_name: string
  lat: number
  lon: number
  radius_km: number
  event_count: number
  source_count: number
  started_at: string
  last_event_at: string
  confirmed: boolean
  incident_name?: string
  events?: EventBrief[]
}

export interface EventBrief {
  id: string
  title: string
  source: string
  severity: string
  category: string
}

export interface CorrelationListResponse {
  correlations: CorrelationFlash[]
  total: number
}

export interface NewsItem {
  id: number
  title: string
  url: string
  description: string
  source_name: string
  source_category: string
  pub_date: string
  ingested_at: string
  relevance_score: number
  lat?: number
  lon?: number
  matched_event_id?: number
  truth_score: number
}

export interface NewsResponse {
  items: NewsItem[]
  total: number
}

export interface IntelBriefingSection {
  title: string
  content: string
}

export interface IntelBriefing {
  content: string
  sections: IntelBriefingSection[]
  generated_at: string
  type: string
  event_count: number
  window_hours: number
}

export interface FinancialOverview {
  vix: number | null
  btc_usd: number | null
  eth_usd: number | null
  oil_wti: number | null
  gold: number | null
  yield_10y: number | null
  yield_2y: number | null
  curve_inverted: boolean | null
  fear_greed: number | null
  timestamp: string
}

export interface EntitySearchResult {
  id: string
  type: string
  name: string
  source: string
  lat?: number
  lon?: number
  last_seen?: string
}

export interface EntitySearchResponse {
  query: string
  results: EntitySearchResult[]
  total: number
}

export interface OSINTResource {
  id: number
  platform: string
  category: string
  display_name: string
  profile_url: string
  description: string
  credibility: string
  is_builtin: boolean
  last_updated: string
  created_at: string
  tags: string[]
  api_key_required: boolean
  free_tier: boolean
  notes: string
  contextual_url?: string
  icon?: string
  label?: string
}

export interface OSINTResourceListResponse {
  resources: OSINTResource[]
  total: number
  limit: number
  offset: number
}

export interface NotificationChannelConfig {
  enabled: boolean
  min_severity: string
  configured: boolean
}

export interface NotificationConfig {
  telegram: NotificationChannelConfig
  slack: NotificationChannelConfig
  discord: NotificationChannelConfig
  email: NotificationChannelConfig
  ntfy: NotificationChannelConfig
}

export interface ProximityConfig {
  configured: boolean
  lat: number
  lon: number
  radius_km: number
}

export interface ProximityEvent {
  event: SentinelEvent
  distance_km: number
}

export interface ProximityEventsResponse {
  events: ProximityEvent[]
  total: number
  radius_km: number
  home_lat: number
  home_lon: number
  configured: boolean
}

export interface UIConfig {
  version: string
  features: Record<string, boolean>
  ui: {
    default_view: string
    default_preset: string
    data_retention_days: number
    sound_enabled: boolean
    sound_volume: number
    ticker_enabled: boolean
    ticker_speed: string
    ticker_min_severity: string
  }
}
