import { useState, useEffect } from 'react'
import { fetchFinancialOverview } from '../../api/client'
import type { FinancialOverview } from '../../types/sentinel'
import { TrendingUp, TrendingDown, DollarSign, Fuel, Clock, AlertTriangle, BarChart3, Gauge } from 'lucide-react'

function formatDollars(value: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value)
}

function formatPercent(value: number): string {
  return `${value.toFixed(2)}%`
}

function vixColor(value: number): string {
  if (value < 20) return 'text-emerald-400'
  if (value <= 30) return 'text-yellow-400'
  return 'text-red-400'
}

function vixBgColor(value: number): string {
  if (value < 20) return 'bg-emerald-400/10 border-emerald-400/20'
  if (value <= 30) return 'bg-yellow-400/10 border-yellow-400/20'
  return 'bg-red-400/10 border-red-400/20'
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

function NullValue() {
  return <span className="text-gray-500 text-sm">N/A</span>
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

export function FinancialDashboard() {
  const [data, setData] = useState<FinancialOverview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let timer: ReturnType<typeof setInterval>

    async function load() {
      try {
        const overview = await fetchFinancialOverview()
        setData(overview)
        setError(null)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load financial data')
      } finally {
        setLoading(false)
      }
    }

    load()
    timer = setInterval(load, 60000)
    return () => clearInterval(timer)
  }, [])

  if (loading) {
    return (
      <div className="p-6">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4">Financial Overview</h2>
        <p className="text-sm text-gray-500">Loading market data...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider mb-4">Financial Overview</h2>
        <div className="bg-red-900/30 border border-red-800 rounded-lg p-3 text-sm text-red-300 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 shrink-0" />
          {error}
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-gray-300 uppercase tracking-wider">Financial Overview</h2>
        {data && (
          <div className="flex items-center gap-1.5 text-xs text-gray-500">
            <Clock className="w-3 h-3" />
            {new Date(data.timestamp).toLocaleString()}
          </div>
        )}
      </div>

      {data && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {/* VIX */}
          <IndicatorCard
            label="VIX (Fear Gauge)"
            icon={<Gauge className="w-4 h-4 text-gray-400" />}
            className={data.vix !== null ? vixBgColor(data.vix) : ''}
          >
            {data.vix !== null ? (
              <p className={`text-2xl font-bold ${vixColor(data.vix)}`}>
                {data.vix.toFixed(2)}
              </p>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* BTC/USD */}
          <IndicatorCard
            label="BTC / USD"
            icon={<DollarSign className="w-4 h-4 text-orange-400" />}
          >
            {data.btc_usd !== null ? (
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
            {data.eth_usd !== null ? (
              <p className="text-2xl font-bold text-gray-100">{formatDollars(data.eth_usd)}</p>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* Oil WTI */}
          <IndicatorCard
            label="Oil WTI"
            icon={<Fuel className="w-4 h-4 text-gray-400" />}
          >
            {data.oil_wti !== null ? (
              <div>
                <p className="text-2xl font-bold text-gray-100">{formatDollars(data.oil_wti)}</p>
                <p className="text-xs text-gray-500 mt-0.5">per barrel</p>
              </div>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* Gold */}
          <IndicatorCard
            label="Gold"
            icon={<DollarSign className="w-4 h-4 text-yellow-400" />}
          >
            {data.gold !== null ? (
              <div>
                <p className="text-2xl font-bold text-gray-100">{formatDollars(data.gold)}</p>
                <p className="text-xs text-gray-500 mt-0.5">per oz</p>
              </div>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* 10Y Treasury */}
          <IndicatorCard
            label="10Y Treasury Yield"
            icon={<BarChart3 className="w-4 h-4 text-gray-400" />}
          >
            {data.yield_10y !== null ? (
              <p className="text-2xl font-bold text-gray-100">{formatPercent(data.yield_10y)}</p>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* 2Y Treasury */}
          <IndicatorCard
            label="2Y Treasury Yield"
            icon={<BarChart3 className="w-4 h-4 text-gray-400" />}
          >
            {data.yield_2y !== null ? (
              <p className="text-2xl font-bold text-gray-100">{formatPercent(data.yield_2y)}</p>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* Yield Curve */}
          <IndicatorCard
            label="Yield Curve"
            icon={data.curve_inverted ? (
              <TrendingDown className="w-4 h-4 text-red-400" />
            ) : (
              <TrendingUp className="w-4 h-4 text-emerald-400" />
            )}
            className={data.curve_inverted === true
              ? 'bg-red-400/10 border-red-400/20'
              : data.curve_inverted === false
                ? 'bg-emerald-400/10 border-emerald-400/20'
                : ''
            }
          >
            {data.curve_inverted !== null ? (
              <div>
                <p className={`text-2xl font-bold ${data.curve_inverted ? 'text-red-400' : 'text-emerald-400'}`}>
                  {data.curve_inverted ? 'INVERTED' : 'NORMAL'}
                </p>
                {data.curve_inverted && (
                  <p className="text-xs text-red-300 mt-0.5 flex items-center gap-1">
                    <AlertTriangle className="w-3 h-3" />
                    Recession signal
                  </p>
                )}
              </div>
            ) : (
              <NullValue />
            )}
          </IndicatorCard>

          {/* Fear & Greed Index */}
          <IndicatorCard
            label="Fear & Greed Index"
            icon={<Gauge className="w-4 h-4 text-gray-400" />}
            className="sm:col-span-2 lg:col-span-1"
          >
            {data.fear_greed !== null ? (
              <div>
                <div className="flex items-baseline gap-2">
                  <p className={`text-2xl font-bold ${fearGreedColor(data.fear_greed)}`}>
                    {data.fear_greed}
                  </p>
                  <span className={`text-sm font-medium ${fearGreedColor(data.fear_greed)}`}>
                    {fearGreedLabel(data.fear_greed)}
                  </span>
                </div>
                <div className="mt-2 w-full bg-gray-700 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full transition-all ${fearGreedBarColor(data.fear_greed)}`}
                    style={{ width: `${data.fear_greed}%` }}
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
      )}
    </div>
  )
}
