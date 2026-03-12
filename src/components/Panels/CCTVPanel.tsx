import { useState, useEffect, useCallback } from 'react'
import { X, RefreshCw, MapPin } from 'lucide-react'
import type { CCTVCamera } from '../../types/livedata'

interface CCTVPanelProps {
  camera: CCTVCamera
  onClose: () => void
}

export default function CCTVPanel({ camera, onClose }: CCTVPanelProps) {
  const [imgSrc, setImgSrc] = useState(camera.image_url)
  const [lastRefresh, setLastRefresh] = useState(Date.now())
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(() => {
    const sep = camera.image_url.includes('?') ? '&' : '?'
    setImgSrc(`${camera.image_url}${sep}_t=${Date.now()}`)
    setLastRefresh(Date.now())
    setLoading(true)
  }, [camera.image_url])

  // Auto-refresh every 30 seconds
  useEffect(() => {
    const interval = setInterval(refresh, 30_000)
    return () => clearInterval(interval)
  }, [refresh])

  // Reset when camera changes
  useEffect(() => {
    setImgSrc(camera.image_url)
    setLastRefresh(Date.now())
    setLoading(true)
  }, [camera.id, camera.image_url])

  return (
    <div className="absolute top-4 right-4 z-40 w-[420px] bg-gray-950/95 border border-gray-800 rounded-lg backdrop-blur-sm shadow-2xl overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-800">
        <div className="flex items-center gap-2 min-w-0">
          <div className="w-2 h-2 rounded-full bg-green-500 shrink-0" style={{ boxShadow: '0 0 6px #22c55e' }} />
          <span className="text-[11px] font-mono text-cyan-400 truncate">{camera.name}</span>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={refresh}
            className="p-1 rounded hover:bg-gray-800 text-gray-500 hover:text-gray-300 transition-colors"
            title="Refresh"
          >
            <RefreshCw size={12} />
          </button>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-gray-800 text-gray-500 hover:text-gray-300 transition-colors"
            title="Close"
          >
            <X size={14} />
          </button>
        </div>
      </div>

      {/* Image */}
      <div className="relative bg-gray-900 aspect-video">
        {loading && (
          <div className="absolute inset-0 flex items-center justify-center">
            <RefreshCw size={16} className="text-gray-600 animate-spin" />
          </div>
        )}
        <img
          src={imgSrc}
          alt={camera.name}
          className="w-full h-full object-cover"
          onLoad={() => setLoading(false)}
          onError={() => setLoading(false)}
        />
        {/* Live badge */}
        <div className="absolute top-2 left-2 flex items-center gap-1 bg-gray-950/80 px-2 py-0.5 rounded">
          <div className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse" />
          <span className="text-[9px] font-mono uppercase text-red-400">LIVE</span>
        </div>
      </div>

      {/* Info footer */}
      <div className="px-3 py-2 border-t border-gray-800 space-y-1">
        <div className="flex items-center gap-2">
          <MapPin size={10} className="text-gray-600 shrink-0" />
          <span className="text-[10px] font-mono text-gray-400">{camera.city}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-[10px] font-mono text-gray-600">
            {camera.lat.toFixed(4)}, {camera.lng.toFixed(4)}
          </span>
          <span className="text-[9px] font-mono text-gray-700">
            {camera.feed_type} &middot; refresh {Math.round((Date.now() - lastRefresh) / 1000)}s ago
          </span>
        </div>
      </div>
    </div>
  )
}
