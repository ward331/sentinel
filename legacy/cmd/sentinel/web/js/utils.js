/* ============================================================
   SENTINEL V3 — Shared Utilities
   ============================================================ */

const Utils = {
  // Time formatting
  timeAgo(dateStr) {
    if (!dateStr) return '';
    const now = Date.now();
    const then = new Date(dateStr).getTime();
    const diff = Math.max(0, now - then);
    const secs = Math.floor(diff / 1000);
    if (secs < 60) return `${secs}s ago`;
    const mins = Math.floor(secs / 60);
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    const days = Math.floor(hrs / 24);
    return `${days}d ago`;
  },

  formatTime(dateStr) {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  },

  formatDate(dateStr) {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
  },

  formatDateTime(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' +
           d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  },

  // Severity helpers
  severityClass(severity) {
    const s = (severity || 'info').toLowerCase();
    const map = { info: 'info', watch: 'watch', warning: 'warning', alert: 'alert', critical: 'critical' };
    return map[s] || 'info';
  },

  severityLabel(severity) {
    const s = (severity || 'info').toLowerCase();
    return s.charAt(0).toUpperCase() + s.slice(1);
  },

  // Truth score display
  truthScoreSymbol(score) {
    const symbols = ['', '\u2460', '\u2461', '\u2462', '\u2463', '\u2464'];
    return symbols[score] || '';
  },

  // Domain helpers
  domainClass(domain) {
    const d = (domain || '').toLowerCase();
    const map = {
      military: 'military', cyber: 'cyber', financial: 'financial',
      natural: 'natural', health: 'health'
    };
    return map[d] || 'natural';
  },

  threatLevelLabel(level) {
    const labels = ['NOMINAL', 'LOW', 'GUARDED', 'ELEVATED', 'HIGH', 'CRITICAL'];
    return labels[level] || 'UNKNOWN';
  },

  // Fetch wrapper
  async apiFetch(url, options = {}) {
    try {
      const resp = await fetch(url, {
        headers: { 'Accept': 'application/json', ...options.headers },
        ...options
      });
      if (!resp.ok) {
        console.error(`API ${resp.status}: ${url}`);
        return null;
      }
      return await resp.json();
    } catch (err) {
      console.error(`API error: ${url}`, err);
      return null;
    }
  },

  // DOM helpers
  el(tag, attrs = {}, children = []) {
    const elem = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs)) {
      if (k === 'className') elem.className = v;
      else if (k === 'innerHTML') elem.innerHTML = v;
      else if (k === 'textContent') elem.textContent = v;
      else if (k.startsWith('on')) elem.addEventListener(k.slice(2).toLowerCase(), v);
      else if (k === 'style' && typeof v === 'object') Object.assign(elem.style, v);
      else elem.setAttribute(k, v);
    }
    for (const child of children) {
      if (typeof child === 'string') elem.appendChild(document.createTextNode(child));
      else if (child) elem.appendChild(child);
    }
    return elem;
  },

  // Simple HTML escape
  esc(str) {
    if (!str) return '';
    const d = document.createElement('div');
    d.textContent = str;
    return d.innerHTML;
  },

  // Debounce
  debounce(fn, ms) {
    let timer;
    return (...args) => {
      clearTimeout(timer);
      timer = setTimeout(() => fn(...args), ms);
    };
  },

  // IndexedDB cache for offline
  async openCache() {
    return new Promise((resolve, reject) => {
      const req = indexedDB.open('sentinel-cache', 1);
      req.onupgradeneeded = () => {
        const db = req.result;
        if (!db.objectStoreNames.contains('events')) {
          db.createObjectStore('events', { keyPath: 'id' });
        }
        if (!db.objectStoreNames.contains('kv')) {
          db.createObjectStore('kv', { keyPath: 'key' });
        }
      };
      req.onsuccess = () => resolve(req.result);
      req.onerror = () => reject(req.error);
    });
  },

  async cacheEvents(events) {
    try {
      const db = await this.openCache();
      const tx = db.transaction('events', 'readwrite');
      const store = tx.objectStore('events');
      // Keep last 100
      const all = await new Promise(r => { const req = store.getAllKeys(); req.onsuccess = () => r(req.result); });
      if (all.length > 100) {
        const toDelete = all.slice(0, all.length - 100);
        for (const key of toDelete) store.delete(key);
      }
      for (const ev of events) store.put(ev);
    } catch (e) {
      // IndexedDB not available
    }
  },

  async getCachedEvents() {
    try {
      const db = await this.openCache();
      const tx = db.transaction('events', 'readonly');
      const store = tx.objectStore('events');
      return new Promise(r => { const req = store.getAll(); req.onsuccess = () => r(req.result || []); });
    } catch (e) {
      return [];
    }
  },

  async cacheKV(key, value) {
    try {
      const db = await this.openCache();
      const tx = db.transaction('kv', 'readwrite');
      tx.objectStore('kv').put({ key, value, ts: Date.now() });
    } catch (e) {}
  },

  async getKV(key) {
    try {
      const db = await this.openCache();
      const tx = db.transaction('kv', 'readonly');
      return new Promise(r => {
        const req = tx.objectStore('kv').get(key);
        req.onsuccess = () => r(req.result ? req.result.value : null);
      });
    } catch (e) {
      return null;
    }
  }
};
