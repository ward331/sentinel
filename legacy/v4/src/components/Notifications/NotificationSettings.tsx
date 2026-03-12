import { useState, useEffect, useCallback } from 'react';
import {
  Send, Hash, MessageCircle, Mail, Bell,
  Loader2, AlertCircle, CheckCircle2, X,
} from 'lucide-react';
import {
  fetchNotificationConfig, updateNotificationConfig, testNotificationChannel,
} from '../../api/client';
import type { NotificationConfig, NotificationChannelConfig } from '../../types/sentinel';

type Severity = 'low' | 'medium' | 'high' | 'critical';
type ChannelKey = keyof NotificationConfig;

interface ChannelDef {
  key: ChannelKey;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}

const CHANNELS: ChannelDef[] = [
  { key: 'telegram', label: 'Telegram', icon: Send },
  { key: 'slack', label: 'Slack', icon: Hash },
  { key: 'discord', label: 'Discord', icon: MessageCircle },
  { key: 'email', label: 'Email', icon: Mail },
  { key: 'ntfy', label: 'Ntfy', icon: Bell },
];

const SEVERITIES: Severity[] = ['low', 'medium', 'high', 'critical'];

const DEFAULT_CHANNEL: NotificationChannelConfig = { enabled: false, min_severity: 'medium', configured: false };

interface Toast { id: number; type: 'success' | 'error'; message: string; }

export function NotificationSettings() {
  const [config, setConfig] = useState<NotificationConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [testingChannel, setTestingChannel] = useState<string | null>(null);
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((type: 'success' | 'error', message: string) => {
    const id = Date.now();
    setToasts(prev => [...prev, { id, type, message }]);
    setTimeout(() => setToasts(prev => prev.filter(t => t.id !== id)), 4000);
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchNotificationConfig()
      .then(data => { if (!cancelled) setConfig(data); })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  const handleToggle = (key: ChannelKey) => {
    if (!config) return;
    const ch = config[key] || DEFAULT_CHANNEL;
    setConfig({ ...config, [key]: { ...ch, enabled: !ch.enabled } });
  };

  const handleSeverityChange = (key: ChannelKey, severity: Severity) => {
    if (!config) return;
    const ch = config[key] || DEFAULT_CHANNEL;
    setConfig({ ...config, [key]: { ...ch, min_severity: severity } });
  };

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    try {
      await updateNotificationConfig(config);
      addToast('success', 'Notification settings saved');
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : 'Failed to save');
    } finally { setSaving(false); }
  };

  const handleTest = async (key: string) => {
    setTestingChannel(key);
    try {
      const result = await testNotificationChannel(key);
      if (result.status === 'sent') addToast('success', `${key} test sent successfully`);
      else addToast('error', result.message || `${key} test failed`);
    } catch (err) {
      addToast('error', err instanceof Error ? err.message : `${key} test failed`);
    } finally { setTestingChannel(null); }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8 bg-gray-900 rounded-lg">
        <Loader2 className="w-5 h-5 text-gray-400 animate-spin" />
        <span className="ml-2 text-gray-400">Loading notification settings...</span>
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

  if (!config) return null;

  return (
    <div className="relative bg-gray-900 rounded-lg p-5 space-y-4">
      <div className="space-y-3">
        {CHANNELS.map(({ key, label, icon: Icon }) => {
          const ch = config[key] || DEFAULT_CHANNEL;
          return (
            <div key={key} className="flex items-center gap-3 bg-gray-800 rounded-lg px-4 py-3">
              <Icon className="w-5 h-5 text-gray-400 shrink-0" />
              <div className="flex items-center gap-2 min-w-[110px]">
                <span className="text-gray-100 font-medium">{label}</span>
                <span className={`w-2 h-2 rounded-full ${ch.configured ? 'bg-emerald-400' : 'bg-gray-500'}`}
                      title={ch.configured ? 'Configured' : 'Not configured'} />
              </div>
              <button type="button" role="switch" aria-checked={!!ch.enabled}
                onClick={() => handleToggle(key)}
                className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors ${ch.enabled ? 'bg-emerald-400' : 'bg-gray-600'}`}>
                <span className={`inline-block h-5 w-5 transform rounded-full bg-white shadow transition-transform mt-0.5 ${ch.enabled ? 'translate-x-5 ml-0.5' : 'translate-x-0.5'}`} />
              </button>
              <select value={ch.min_severity || 'medium'}
                onChange={e => handleSeverityChange(key, e.target.value as Severity)}
                className="bg-gray-700 text-gray-100 text-sm rounded px-2 py-1 border border-gray-600 focus:outline-none focus:border-emerald-400">
                {SEVERITIES.map(s => <option key={s} value={s}>{s.charAt(0).toUpperCase() + s.slice(1)}+</option>)}
              </select>
              <button onClick={() => handleTest(key)} disabled={testingChannel === key}
                className="ml-auto px-3 py-1 text-sm rounded bg-gray-700 text-gray-300 hover:bg-gray-600 hover:text-gray-100 disabled:opacity-50 transition-colors">
                {testingChannel === key ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Test'}
              </button>
            </div>
          );
        })}
      </div>

      <div className="flex justify-end pt-2">
        <button onClick={handleSave} disabled={saving}
          className="flex items-center gap-2 px-4 py-2 rounded bg-emerald-500 hover:bg-emerald-400 text-gray-950 font-medium text-sm disabled:opacity-50 transition-colors">
          {saving && <Loader2 className="w-4 h-4 animate-spin" />}
          Save Settings
        </button>
      </div>

      {toasts.length > 0 && (
        <div className="fixed bottom-4 right-4 z-50 space-y-2">
          {toasts.map(toast => (
            <div key={toast.id} className={`flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-sm ${
              toast.type === 'success' ? 'bg-emerald-400/15 border border-emerald-400/30 text-emerald-400'
                : 'bg-red-400/15 border border-red-400/30 text-red-400'}`}>
              {toast.type === 'success' ? <CheckCircle2 className="w-4 h-4 shrink-0" /> : <AlertCircle className="w-4 h-4 shrink-0" />}
              <span>{toast.message}</span>
              <button onClick={() => setToasts(p => p.filter(t => t.id !== toast.id))} className="ml-2 opacity-60 hover:opacity-100">
                <X className="w-3 h-3" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
