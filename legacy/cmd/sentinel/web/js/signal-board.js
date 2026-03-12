/* ============================================================
   SENTINEL V3 — Signal Board View
   ============================================================ */

const SignalBoard = {
  data: null,

  async render(container) {
    container.innerHTML = `
      <div class="section-header">
        <span class="section-header__title">Signal Board</span>
        <span id="sb-updated" style="font-size:11px;color:var(--text-muted)"></span>
      </div>
      <div id="sb-domains" class="domain-grid"></div>
      <div id="sb-stats" class="signal-stats"></div>
      <div class="section-header" style="margin-top:8px">
        <span class="section-header__title">Critical Events</span>
      </div>
      <div id="sb-critical"></div>
    `;
    await this.load();
  },

  async load() {
    const data = await Utils.apiFetch('/api/signal-board');
    if (data) {
      this.data = data;
      Utils.cacheKV('signal-board', data);
    } else {
      this.data = await Utils.getKV('signal-board');
    }
    this.update();
  },

  update() {
    const d = this.data;
    if (!d) {
      document.getElementById('sb-domains').innerHTML = '<div class="loading">Waiting for signal data</div>';
      return;
    }

    // Domains
    const domains = d.domains || [
      { name: 'Military', level: 0 },
      { name: 'Cyber', level: 0 },
      { name: 'Financial', level: 0 },
      { name: 'Natural', level: 0 },
      { name: 'Health', level: 0 }
    ];

    const domainEl = document.getElementById('sb-domains');
    domainEl.innerHTML = domains.map(dom => {
      const cls = Utils.domainClass(dom.name);
      const level = dom.level || dom.threat_level || 0;
      return `
        <div class="domain-card domain-card--${cls}" onclick="App.navigate('#/feed?category=${dom.name.toLowerCase()}')">
          <div class="domain-card__name">${Utils.esc(dom.name)}</div>
          <div class="domain-card__level threat-${level}">${level}</div>
          <div class="domain-card__label">${Utils.threatLevelLabel(level)}</div>
        </div>
      `;
    }).join('');

    // Stats
    const statsEl = document.getElementById('sb-stats');
    const activeAlerts = d.active_alerts || d.activeAlerts || 0;
    const correlations = d.active_correlations || d.activeCorrelations || 0;
    const totalEvents = d.total_events || d.totalEvents || 0;
    statsEl.innerHTML = `
      <div class="signal-stat">
        <div class="signal-stat__value">${activeAlerts}</div>
        <div class="signal-stat__label">Active Alerts</div>
      </div>
      <div class="signal-stat">
        <div class="signal-stat__value">${correlations}</div>
        <div class="signal-stat__label">Correlations</div>
      </div>
      <div class="signal-stat">
        <div class="signal-stat__value">${totalEvents}</div>
        <div class="signal-stat__label">Events (24h)</div>
      </div>
    `;

    // Critical events
    const critEl = document.getElementById('sb-critical');
    const criticals = (d.critical_events || d.criticalEvents || d.recent_critical || []).slice(0, 5);
    if (criticals.length === 0) {
      critEl.innerHTML = '<div class="empty-state"><div class="empty-state__text">No critical events</div></div>';
    } else {
      critEl.innerHTML = criticals.map(ev => `
        <div class="event-card card--clickable" onclick="App.navigate('#/feed?id=${ev.id}')">
          <div class="event-card__severity event-card__severity--critical"></div>
          <div class="event-card__body">
            <div class="event-card__title">${Utils.esc(ev.title || ev.description || '')}</div>
            <div class="event-card__meta">
              <span class="event-card__source">${Utils.esc(ev.source || '')}</span>
              <span>${Utils.timeAgo(ev.timestamp || ev.created_at)}</span>
            </div>
          </div>
        </div>
      `).join('');
    }

    // Update timestamp
    const updEl = document.getElementById('sb-updated');
    if (updEl) updEl.textContent = 'Updated ' + new Date().toLocaleTimeString();
  },

  onSSE(eventData) {
    if (eventData && eventData.type === 'signal_board') {
      this.data = eventData.data || eventData;
      this.update();
    }
  }
};
