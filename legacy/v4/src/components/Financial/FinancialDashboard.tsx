import { useState, useEffect } from 'react'
import type { FinancialData } from '../../types/livedata'
import { TrendingUp, TrendingDown, DollarSign, Clock, AlertTriangle, Gauge } from 'lucide-react'

function formatDollars(value: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value)
}

function fearGreedLabel(value: number): string {
  if (value <= 25) return 'Extreme Fear'
  if (value <= 45) return 'Fear'
  if (value <= 55) return 'Neutral'
  if (value <= 75) return 'Greed'
  return 'Extreme Greed'
}

function fearGreedColor(value: number): string {
  if (value <= 25) return 'text-red-400'
  if (value <= 45) return 'text-orange-400'
  if (value <= 55) return 'text-gray-400'
  if (value <= 75) return 'text-yellow-300'
  return 'text-emerald-400'
}

function fearGreedBarColor(value: number): string {
  if (value <= 25) return 'bg-red-400'
  if (value <= 45) return 'bg-orange-400'
  if (value <= 55) return 'bg-gray-400'
  if (value <= 75) return 'bg-yellow-300'
  return 'bg-emerald-400'
}

interface IndicatorCardProps {
  label: string
  icon: React.ReactNode
  children: React.ReactNode
  className?: string
}

function IndicatorCard({ label, icon, children, className = '' }: IndicatorCardProps) {
  return (
    <div className={`bg-gray-800 rounded-lg border border-gray-700/50 p-4 ${className}`}>
      <div className="flex items-center gap-2 mb-2">
        {icon}
        <span className="text-xs font-semibold text-gray-400 uppercase tracking-wider">{label}</span>
      </div>
      {children}
    </div>
  )
}

function NullValue() {
  return <span className="text-gray-500 text-sm">N/A</span>
}

interface FinancialDashboardProps {
  data?: FinancialData | null
}

export function FinancialDashboard({ data }: FinancialDashboardProps) {
  const [now, setNow] = useState(new Date())

  // Update clock every 60s
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 60000)
    return () => clearInterval(t)
  }, [])

  if (!data) {
    return (
      <div className="p-6">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4">Financial Overview</h2>
        <div className="bg-gray-900 rounded-lg border border-gray-800 p-8 text-center">
          <DollarSign className="w-8 h-8 text-gray-600 mx-auto mb-3" />
          <p className="text-sm text-gray-500">Waiting for financial data from OSINT backend...</p>
          <p className="text-xs text-gray-600 mt-1">Data refreshes every 2 minutes</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Financial Overview</h2>
        <div className="flex items-center gap-1.5 text-xs text-gray-500">
          <Clock className="w-3 h-3" />
          {now.toLocaleString()}
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {/* BTC/USD */}
        <IndicatorCard
          label="BTC / USD"
          icon={<DollarSign className="w-4 h-4 text-orange-400" />}
        >
          {data.btc_usd != null ? (
            <p className="text-2xl font-bold text-gray-100">{formatDollars(data.btc_usd)}</p>
          ) : (
            <NullValue />
          )}
        </IndicatorCard>

        {/* ETH/USD */}
        <IndicatorCard
          label="ETH / USD"
          icon={<DollarSign className="w-4 h-4 text-blue-400" />}
        >
          {data.eth_usd != null ? (
            <p className="text-2xl font-bold text-gray-100">{formatDollars(data.eth_usd)}</p>
          ) : (
            <NullValue />
          )}
        </IndicatorCard>

        {/* Fear & Greed Index */}
        <IndicatorCard
          label="Fear & Greed Index"
          icon={<Gauge className="w-4 h-4 text-gray-400" />}
        >
          {data.fear_greed_index != null ? (
            <div>
              <div className="flex items-baseline gap-2">
                <p className={`text-2xl font-bold ${fearGreedColor(data.fear_greed_index)}`}>
                  {data.fear_greed_index}
                </p>
                <span className={`text-sm font-medium ${fearGreedColor(data.fear_greed_index)}`}>
                  {data.fear_greed_label || fearGreedLabel(data.fear_greed_index)}
                </span>
              </div>
              <div className="mt-2 w-full bg-gray-700 rounded-full h-2">
                <div
                  className={`h-2 rounded-full transition-all ${fearGreedBarColor(data.fear_greed_index)}`}
                  style={{ width: `${data.fear_greed_index}%` }}
                />
              </div>
              <div className="flex justify-between mt-1 text-[10px] text-gray-500">
                <span>0 Fear</span>
                <span>50</span>
                <span>100 Greed</span>
              </div>
            </div>
          ) : (
            <NullValue />
          )}
        </IndicatorCard>
      </div>

      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4 mt-4">
        <p className="text-xs text-gray-500">
          Data sourced from CoinGecko (crypto prices) and Alternative.me (Fear &amp; Greed Index).
          Refreshes every 2 minutes via OSINT backend.
        </p>
      </div>
    </div>
  )
}
