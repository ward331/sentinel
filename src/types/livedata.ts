export interface Aircraft {
  icao: string
  callsign: string
  lat: number
  lon: number
  alt_ft: number
  speed_kts: number
  heading: number
  on_ground: boolean
  category: 'commercial' | 'military' | 'private'
  squawk?: string
  trail?: [number, number][]
}

export interface Vessel {
  mmsi: string
  name: string
  lat: number
  lon: number
  speed: number
  course: number
  ship_type: 'cargo' | 'tanker' | 'passenger' | 'military' | 'fishing' | 'pleasure' | 'unknown'
  destination?: string
  flag?: string
}

export interface Satellite {
  name: string
  norad_id: number
  lat: number
  lon: number
  alt_km: number
  speed_kph: number
  mission_type: 'military_recon' | 'sar' | 'sigint' | 'navigation' | 'early_warning' | 'commercial' | 'iss'
  country?: string
}

export interface Earthquake {
  id: string
  mag: number
  place: string
  lat: number
  lon: number
  depth_km: number
  time: number
  url: string
}

export interface Fire {
  lat: number
  lon: number
  brightness: number
  frp: number
  acq_date: string
  confidence: string
}

export interface GdeltEvent {
  title: string
  lat: number
  lon: number
  tone: number
  url: string
  domain: string
  date: string
}

export interface KiwiSDR {
  name: string
  lat: number
  lon: number
  url: string
  bands?: string
  users_active?: number
}

export interface SpaceWeather {
  kp_index: number
  timestamp: string
  storm_level: 'quiet' | 'active' | 'storm'
}

export interface FinancialData {
  fear_greed_index?: number
  fear_greed_label?: string
  btc_usd?: number
  eth_usd?: number
}

export interface LiveData {
  commercial_flights: Aircraft[]
  military_flights: Aircraft[]
  private_flights: Aircraft[]
  ships: Vessel[]
  satellites: Satellite[]
  earthquakes: Earthquake[]
  firms_fires: Fire[]
  gdelt: GdeltEvent[]
  kiwisdr: KiwiSDR[]
  space_weather?: SpaceWeather
  internet_outages?: Array<{ country: string; score: number; normally: number }>
  datacenters?: Array<{ name: string; operator: string; lat: number; lon: number; region: string }>
  financial?: FinancialData
  news?: Array<{ title: string; link: string; source: string; published: string; summary: string; lat?: number; lon?: number }>
  freshness?: Record<string, string>
}
