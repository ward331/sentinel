import { useState } from 'react'
import { BarChart3, ChevronDown, ChevronUp, TrendingUp, TrendingDown } from 'lucide-react'
import type { FinancialData } from '../../types/livedata'

interface MarketsPanelProps {
  financial: FinancialData | null
}

function fearGreedColor(value: number): string {
  if (value <= 20) return '#ef4444'
  if (value <= 40) return '#f97316'
  if (value <= 60) return '#eab308'
  if (value <= 80) return '#22c55e'
  return '#10b981'
}

function formatPrice(value: number | undefined): string {
  if (value === undefined) return '--'
  if (value >= 1000) return value.toLocaleString('en-US', { maximumFractionDigits: 0 })
  return value.toFixed(2)
}

export default function MarketsPanel({ financial }: MarketsPanelProps) {
  const [visible, setVisible] = useState(false)

  if (!financial) return null

  const fgValue = financial.fear_greed_index ?? 50
  const fgColor = fearGreedColor(fgValue)

  return (
    <div className="absolute bottom-8 right-4 z-20">
      {/* Toggle */}
      <button
        onClick={() => setVisible(!visible)}
        className="flex items-center gap-1.5 px-2 py-1 bg-gray-950/90 border border-gray-800 rounded text-[10px] font-mono text-gray-500 hover:text-gray-300 hover:border-gray-700 transition-all ml-auto mb-1"
      >
        <BarChart3 size={10} />
        MARKETS
        {visible ? <ChevronDown size={10} /> : <ChevronUp size={10} />}
      </button>

      {visible && (
        <div className="bg-gray-950/95 border border-gray-800 rounded-lg p-3 backdrop-blur-sm w-52">
          {/* Fear & Greed */}
          <div className="mb-3">
            <div className="flex items-center justify-between mb-1">
              <span className="text-[9px] font-mono uppercase text-gray-500">FEAR & GREED</span>
              <span className="text-[10px] font-mono uppercase" style={{ color: fgColor }}>
                {financial.fear_greed_label || (fgValue <= 25 ? 'EXTREME FEAR' : fgValue <= 45 ? 'FEAR' : fgValue <= 55 ? 'NEUTRAL' : fgValue <= 75 ? 'GREED' : 'EXTREME GREED')}
              </span>
            </div>
            {/* Gauge bar */}
            <div className="relative h-2 bg-gray-800 rounded-full overflow-hidden">
              <div
                className="absolute inset-0 rounded-full"
                style={{
                  background: 'linear-gradient(to right, #ef4444, #f97316, #eab308, #22c55e, #10b981)',
                  opacity: 0.3,
                }}
              />
              <div
                className="absolute top-0 h-full w-1 rounded-full transition-all duration-500"
                style={{
                  left: `${fgValue}%`,
                  backgroundColor: fgColor,
                  boxShadow: `0 0 6px ${fgColor}`,
                }}
              />
            </div>
            <div className="flex items-center justify-between mt-0.5">
              <span className="text-[8px] font-mono text-gray-700">0</span>
              <span className="text-sm font-mono font-bold" style={{ color: fgColor }}>
                {fgValue}
              </span>
              <span className="text-[8px] font-mono text-gray-700">100</span>
            </div>
          </div>

          {/* Crypto prices */}
          <div className="space-y-1.5 border-t border-gray-800 pt-2">
            {financial.btc_usd !== undefined && (
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono text-gray-500">BTC</span>
                <div className="flex items-center gap-1">
                  <span className="text-[11px] font-mono text-gray-300">
                    ${formatPrice(financial.btc_usd)}
                  </span>
                  <TrendingUp size={10} className="text-green-600" />
                </div>
              </div>
            )}
            {financial.eth_usd !== undefined && (
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono text-gray-500">ETH</span>
                <div className="flex items-center gap-1">
                  <span className="text-[11px] font-mono text-gray-300">
                    ${formatPrice(financial.eth_usd)}
                  </span>
                  <TrendingDown size={10} className="text-red-600" />
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
