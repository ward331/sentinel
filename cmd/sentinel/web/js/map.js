/* ============================================================
   SENTINEL V3 — Leaflet Map View
   ============================================================ */

const MapView = {
  map: null,
  markers: [],
  markerLayer: null,
  events: [],
  leafletLoaded: false,

  async render(container) {
    container.innerHTML = `
      <div class="map-container" style="margin:-12px;height:calc(100% + 24px)">
        <div id="map" style="width:100%;height:100%;background:var(--bg-primary)"></div>
        <div class="map-controls">
          <button class="btn btn--small" onclick="MapView.nearMe()" title="Center on my location">Near Me</button>
          <button class="btn btn--small" onclick="MapView.toggleTimeline()" title="Time scrubber">Timeline</button>
          <button class="btn btn--small" onclick="MapView.refresh()" title="Refresh events">Refresh</button>
        </div>
      </div>
    `;

    await this.ensureLeaflet();
    this.initMap();
    await this.loadEvents();
  },

  async ensureLeaflet() {
    if (this.leafletLoaded) return;
    if (typeof L !== 'undefined') { this.leafletLoaded = true; return; }

    // Load CSS
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
    document.head.appendChild(link);

    // Load JS
    await new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
      script.onload = resolve;
      script.onerror = reject;
      document.head.appendChild(script);
    });

    this.leafletLoaded = true;
  },

  initMap() {
    if (this.map) {
      this.map.remove();
      this.map = null;
    }

    this.map = L.map('map', {
      center: [20, 0],
      zoom: 2,
      zoomControl: false,
      attributionControl: false
    });

    // Dark tile layer
    L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
      maxZoom: 19,
      subdomains: 'abcd'
    }).addTo(this.map);

    // Zoom control bottom-left
    L.control.zoom({ position: 'bottomleft' }).addTo(this.map);

    // Attribution
    L.control.attribution({ position: 'bottomleft', prefix: false })
      .addAttribution('SENTINEL')
      .addTo(this.map);

    this.markerLayer = L.layerGroup().addTo(this.map);

    // Fix for tile rendering on dynamic containers
    setTimeout(() => this.map.invalidateSize(), 200);
  },

  async loadEvents() {
    const data = await Utils.apiFetch('/api/events?limit=200');
    let events = [];
    if (data) {
      events = Array.isArray(data) ? data : (data.events || data.items || []);
    }
    this.events = events;
    this.plotEvents(events);
  },

  plotEvents(events) {
    if (!this.markerLayer) return;
    this.markerLayer.clearLayers();
    this.markers = [];

    for (const ev of events) {
      const lat = ev.latitude || ev.lat;
      const lng = ev.longitude || ev.lng || ev.lon;
      if (!lat || !lng) continue;

      const color = this.severityColor(ev.severity);
      const marker = L.circleMarker([lat, lng], {
        radius: this.severityRadius(ev.severity),
        color: color,
        fillColor: color,
        fillOpacity: 0.6,
        weight: 1
      });

      marker.bindPopup(`
        <div style="font-family:monospace;font-size:12px;max-width:250px">
          <strong>${Utils.esc(ev.title || ev.description || 'Event')}</strong><br>
          <span style="color:#888">${Utils.esc(ev.source || '')} &middot; ${Utils.timeAgo(ev.timestamp || ev.created_at)}</span><br>
          ${ev.location ? `<span>${Utils.esc(ev.location)}</span><br>` : ''}
          <span style="color:${color};font-weight:600">${Utils.severityLabel(ev.severity)}</span>
        </div>
      `, { className: 'sentinel-popup' });

      this.markerLayer.addLayer(marker);
      this.markers.push({ marker, event: ev });
    }
  },

  severityColor(severity) {
    const map = {
      info: '#3b82f6', watch: '#8b5cf6', warning: '#f59e0b',
      alert: '#f97316', critical: '#dc2626'
    };
    return map[(severity || 'info').toLowerCase()] || '#3b82f6';
  },

  severityRadius(severity) {
    const map = { info: 5, watch: 6, warning: 7, alert: 8, critical: 10 };
    return map[(severity || 'info').toLowerCase()] || 5;
  },

  nearMe() {
    if (!navigator.geolocation) return;
    navigator.geolocation.getCurrentPosition(pos => {
      this.map.setView([pos.coords.latitude, pos.coords.longitude], 8);
      // Draw proximity circle
      L.circle([pos.coords.latitude, pos.coords.longitude], {
        radius: 100000,
        color: '#00d4ff',
        fillColor: '#00d4ff',
        fillOpacity: 0.05,
        weight: 1,
        dashArray: '4'
      }).addTo(this.map);
    });
  },

  toggleTimeline() {
    const bar = document.getElementById('timeline-bar');
    if (bar) {
      bar.classList.toggle('timeline-bar--active');
      Timeline.init();
    }
  },

  async refresh() {
    await this.loadEvents();
  },

  onSSE(eventData) {
    if (eventData && (eventData.type === 'new_event' || eventData.type === 'event')) {
      const ev = eventData.data || eventData;
      const lat = ev.latitude || ev.lat;
      const lng = ev.longitude || ev.lng || ev.lon;
      if (!lat || !lng || !this.markerLayer) return;

      this.events.unshift(ev);
      const color = this.severityColor(ev.severity);
      const marker = L.circleMarker([lat, lng], {
        radius: this.severityRadius(ev.severity),
        color: color,
        fillColor: color,
        fillOpacity: 0.6,
        weight: 1
      });
      marker.bindPopup(`
        <div style="font-family:monospace;font-size:12px;max-width:250px">
          <strong>${Utils.esc(ev.title || ev.description || '')}</strong><br>
          <span style="color:${color}">${Utils.severityLabel(ev.severity)}</span>
        </div>
      `);
      this.markerLayer.addLayer(marker);

      // Pulse effect for new events
      const pulse = L.circleMarker([lat, lng], {
        radius: 20, color: color, fillOpacity: 0, weight: 2, opacity: 0.8
      }).addTo(this.map);
      setTimeout(() => this.map.removeLayer(pulse), 2000);
    }
  },

  destroy() {
    if (this.map) {
      this.map.remove();
      this.map = null;
    }
  }
};
