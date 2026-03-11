import { useState, useEffect, useRef, useCallback } from 'react';
import { MapPin, Loader2, AlertCircle, Crosshair, Clock } from 'lucide-react';
import { fetchProximityConfig, updateProximityConfig, fetchProximityEvents } from '../../api/client';
import type { ProximityConfig, ProximityEvent } from '../../types/sentinel';

type TimeWindow = '30m' | '1h' | '6h' | '24h';

const TIME_WINDOWS: { value: TimeWindow; label: string; minutes: number }[] = [
  { value: '30m', label: '30 min', minutes: 30 },
  { value: '1h', label: '1 hour', minutes: 60 },
  { value: '6h', label: '6 hours', minutes: 360 },
  { value: '24h', label: '24 hours', minutes: 1440 },
];

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-400/20 text-red-400 border-red-400/30',
  high: 'bg-orange-400/20 text-orange-400 border-orange-400/30',
  medium: 'bg-yellow-400/20 text-yellow-400 border-yellow-400/30',
  low: 'bg-emerald-400/20 text-emerald-400 border-emerald-400/30',
};

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function ProximityPanel() {
  const [config, setConfig] = useState<ProximityConfig | null>(null);
  const [lat, setLat] = useState('');
  const [lon, setLon] = useState('');
  const [radiusKm, setRadiusKm] = useState(50);
  const [timeWindow, setTimeWindow] = useState<TimeWindow>('1h');
  const [events, setEvents] = useState<ProximityEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveMsg, setSaveMsg] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchProximityConfig()
      .then(data => {
        if (cancelled) return;
        setConfig(data);
        if (data.lat != null) setLat(String(data.lat));
        if (data.lon != null) setLon(String(data.lon));
        if (data.radius_km != null) setRadiusKm(data.radius_km);
      })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  const isConfigured = config?.configured ?? (config?.lat != null && config?.lon != null);

  const loadEvents = useCallback(async () => {
    if (!isConfigured) return;
    const minutes = TIME_WINDOWS.find(t => t.value === timeWindow)?.minutes ?? 60;
    setEventsLoading(true);
    try {
      const data = await fetchProximityEvents(minutes);
      const sorted = [...(data.events || [])].sort((a, b) => a.distance_km - b.distance_km);
      setEvents(sorted);
    } catch { /* silent */ }
    finally { setEventsLoading(false); }
  }, [isConfigured, timeWindow]);

  useEffect(() => {
    if (!isConfigured) return;
    loadEvents();
    intervalRef.current = setInterval(loadEvents, 30_000);
    return () => { if (intervalRef.current) clearInterval(intervalRef.current); };
  }, [isConfigured, loadEvents]);

  const handleSave = async () => {
    const latNum = parseFloat(lat);
    const lonNum = parseFloat(lon);
    if (isNaN(latNum) || isNaN(lonNum)) { setSaveMsg('Invalid latitude or longitude'); return; }
    setSaving(true); setSaveMsg(null);
    try {
      await updateProximityConfig({ lat: latNum, lon: lonNum, radius_km: radiusKm });
      setConfig({ configured: true, lat: latNum, lon: lonNum, radius_km: radiusKm });
      setSaveMsg('Location saved');
      setTimeout(() => setSaveMsg(null), 3000);
    } catch (err) {
      setSaveMsg(err instanceof Error ? err.message : 'Failed to save');
    } finally { setSaving(false); }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8 bg-gray-900 rounded-lg">
        <Loader2 className="w-5 h-5 text-gray-400 animate-spin" />
        <span className="ml-2 text-gray-400">Loading proximity config...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 p-4 bg-gray-900 rounded-lg border border-red-400/30">
        <AlertCircle className="w-5 h-5 text-red-400 shrink-0" />
        <span className="text-red-400">{error}</span>
      </div>
    );
  }

  return (
    <div className="bg-gray-900 rounded-lg p-5 space-y-5">
      <div className="space-y-3">
        <h3 className="text-base font-semibold text-gray-100 flex items-center gap-2">
          <Crosshair className="w-5 h-5 text-emerald-400" />
          Location
        </h3>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Latitude</label>
            <input type="number" step="any" value={lat} onChange={e => setLat(e.target.value)} placeholder="e.g. 40.7128"
              className="w-full bg-gray-800 text-gray-100 rounded px-3 py-2 text-sm border border-gray-700 focus:outline-none focus:border-emerald-400 placeholder-gray-500" />
          </div>
          <div>
            <label className="block text-sm text-gray-400 mb-1">Longitude</label>
            <input type="number" step="any" value={lon} onChange={e => setLon(e.target.value)} placeholder="e.g. -74.0060"
              className="w-full bg-gray-800 text-gray-100 rounded px-3 py-2 text-sm border border-gray-700 focus:outline-none focus:border-emerald-400 placeholder-gray-500" />
          </div>
        </div>
        <div>
          <label className="block text-sm text-gray-400 mb-1">Radius: <span className="text-gray-100 font-medium">{radiusKm} km</span></label>
          <input type="range" min={10} max={500} step={10} value={radiusKm} onChange={e => setRadiusKm(Number(e.target.value))} className="w-full accent-emerald-400" />
          <div className="flex justify-between text-xs text-gray-500 mt-0.5"><span>10 km</span><span>500 km</span></div>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={handleSave} disabled={saving}
            className="flex items-center gap-2 px-4 py-2 rounded bg-emerald-500 hover:bg-emerald-400 text-gray-950 font-medium text-sm disabled:opacity-50 transition-colors">
            {saving && <Loader2 className="w-4 h-4 animate-spin" />}
            Save Location
          </button>
          {saveMsg && <span className={`text-sm ${saveMsg.includes('saved') ? 'text-emerald-400' : 'text-red-400'}`}>{saveMsg}</span>}
        </div>
      </div>

      <div className="border-t border-gray-800" />

      {!isConfigured ? (
        <div className="flex flex-col items-center justify-center py-8 text-gray-500">
          <MapPin className="w-8 h-8 mb-2" />
          <p className="text-sm">Configure your location to see nearby events</p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5 text-sm text-gray-400">
              <Clock className="w-4 h-4" /><span>Time window</span>
            </div>
            <div className="flex gap-1">
              {TIME_WINDOWS.map(({ value, label }) => (
                <button key={value} onClick={() => setTimeWindow(value)}
                  className={`px-2.5 py-1 text-xs rounded transition-colors ${
                    timeWindow === value ? 'bg-emerald-400/20 text-emerald-400 border border-emerald-400/30'
                      : 'bg-gray-800 text-gray-400 border border-gray-700 hover:text-gray-300'}`}>
                  {label}
                </button>
              ))}
            </div>
          </div>

          {eventsLoading && events.length === 0 ? (
            <div className="flex items-center justify-center py-6">
              <Loader2 className="w-5 h-5 text-gray-400 animate-spin" />
              <span className="ml-2 text-gray-400 text-sm">Loading nearby events...</span>
            </div>
          ) : events.length === 0 ? (
            <div className="text-center py-6 text-gray-500 text-sm">
              No events within {radiusKm} km in the last {TIME_WINDOWS.find(t => t.value === timeWindow)?.label}
            </div>
          ) : (
            <div className="space-y-2">
              <div className="text-xs text-gray-500">
                {events.length} event{events.length !== 1 ? 's' : ''} nearby
                {eventsLoading && <Loader2 className="w-3 h-3 inline ml-1 animate-spin" />}
              </div>
              {events.map((pe, idx) => (
                <div key={pe.event.id ?? idx} className="flex items-start gap-3 bg-gray-800 rounded-lg px-4 py-3">
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-gray-100 truncate">{pe.event.title}</p>
                    <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
                      <span>{pe.distance_km.toFixed(1)} km away</span>
                      <span>&middot;</span>
                      <span>{timeAgo(pe.event.occurred_at)}</span>
                    </div>
                  </div>
                  <span className={`shrink-0 px-2 py-0.5 text-xs rounded border ${SEVERITY_COLORS[pe.event.severity] || SEVERITY_COLORS.low}`}>
                    {pe.event.severity}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
