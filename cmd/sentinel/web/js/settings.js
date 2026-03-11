/* ============================================================
   SENTINEL V3 — Settings & First-Run Wizard
   ============================================================ */

const Settings = {
  providers: [],
  notifConfig: null,

  async render(container) {
    container.innerHTML = `
      <div class="section-header">
        <span class="section-header__title">Settings</span>
      </div>

      <div class="settings-section">
        <div class="settings-section__title">General</div>
        <div class="settings-row">
          <div>
            <div class="settings-row__label">Theme</div>
            <div class="settings-row__desc">Visual appearance</div>
          </div>
          <select class="input" style="width:auto;min-height:36px" disabled>
            <option selected>Dark</option>
          </select>
        </div>
        <div class="settings-row">
          <div>
            <div class="settings-row__label">Units</div>
            <div class="settings-row__desc">Measurement system</div>
          </div>
          <select class="input" style="width:auto;min-height:36px" id="settings-units">
            <option value="metric">Metric</option>
            <option value="imperial">Imperial</option>
          </select>
        </div>
      </div>

      <div class="settings-section">
        <div class="settings-section__title">Notifications</div>
        <div id="settings-notif"></div>
      </div>

      <div class="settings-section">
        <div class="settings-section__title">Providers</div>
        <div id="settings-providers"><div class="loading">Loading providers</div></div>
      </div>

      <div class="settings-section">
        <div class="settings-section__title">About</div>
        <div class="card">
          <div style="text-align:center;padding:12px">
            <div style="font-size:18px;font-weight:700;color:var(--accent);letter-spacing:2px;margin-bottom:4px">SENTINEL</div>
            <div style="font-size:12px;color:var(--text-muted)">v3.0.0</div>
            <div style="font-size:11px;color:var(--text-muted);margin-top:8px">Real-time global intelligence monitoring</div>
            <div style="font-size:10px;color:var(--text-muted);margin-top:12px">MIT License</div>
          </div>
        </div>
      </div>
    `;

    // Load settings
    const savedUnits = localStorage.getItem('sentinel-units') || 'metric';
    document.getElementById('settings-units').value = savedUnits;
    document.getElementById('settings-units').addEventListener('change', e => {
      localStorage.setItem('sentinel-units', e.target.value);
    });

    await Promise.all([this.loadProviders(), this.loadNotifications()]);
  },

  async loadProviders() {
    const data = await Utils.apiFetch('/api/providers');
    const container = document.getElementById('settings-providers');
    if (!data) {
      container.innerHTML = '<div class="empty-state"><div class="empty-state__text">Could not load providers</div></div>';
      return;
    }

    const providers = Array.isArray(data) ? data : (data.providers || []);
    this.providers = providers;

    if (providers.length === 0) {
      container.innerHTML = '<div class="empty-state"><div class="empty-state__text">No providers configured</div></div>';
      return;
    }

    container.innerHTML = providers.map(p => {
      const name = p.name || p.id || 'Unknown';
      const enabled = p.enabled !== false;
      const healthy = p.healthy !== false && p.status !== 'error';
      const degraded = p.status === 'degraded' || p.status === 'slow';
      const statusClass = healthy ? 'healthy' : (degraded ? 'degraded' : 'down');
      const eventsHour = p.events_per_hour || p.eventsPerHour || 0;
      const lastFetch = p.last_fetch || p.lastFetch || p.last_poll || '';

      return `
        <div class="provider-row">
          <div class="provider-row__dot provider-row__dot--${statusClass}"></div>
          <div class="provider-row__info">
            <div class="provider-row__name">${Utils.esc(name)}</div>
            <div class="provider-row__meta">
              ${eventsHour > 0 ? `${eventsHour} events/hr` : ''}
              ${lastFetch ? ` &middot; Last: ${Utils.timeAgo(lastFetch)}` : ''}
            </div>
          </div>
          <label class="toggle">
            <input type="checkbox" ${enabled ? 'checked' : ''} data-provider="${Utils.esc(name)}">
            <span class="toggle__slider"></span>
          </label>
        </div>
      `;
    }).join('');
  },

  async loadNotifications() {
    const data = await Utils.apiFetch('/api/notifications/config');
    const container = document.getElementById('settings-notif');

    const channels = [
      { id: 'telegram', name: 'Telegram', desc: 'Bot notifications via Telegram' },
      { id: 'email', name: 'Email', desc: 'Email alerts' },
      { id: 'ntfy', name: 'ntfy', desc: 'Push via ntfy.sh' },
      { id: 'discord', name: 'Discord', desc: 'Webhook to Discord channel' },
      { id: 'webhook', name: 'Webhook', desc: 'Custom HTTP webhook' }
    ];

    const config = data || {};

    container.innerHTML = channels.map(ch => {
      const chConfig = config[ch.id] || {};
      const enabled = chConfig.enabled || false;
      return `
        <div class="settings-row">
          <div>
            <div class="settings-row__label">${ch.name}</div>
            <div class="settings-row__desc">${ch.desc}</div>
          </div>
          <div style="display:flex;align-items:center;gap:8px">
            <button class="btn btn--small" onclick="Settings.testChannel('${ch.id}')">Test</button>
            <label class="toggle">
              <input type="checkbox" ${enabled ? 'checked' : ''} data-channel="${ch.id}">
              <span class="toggle__slider"></span>
            </label>
          </div>
        </div>
      `;
    }).join('');
  },

  async testChannel(channel) {
    const result = await Utils.apiFetch(`/api/notifications/test/${channel}`, { method: 'POST' });
    if (result && result.success) {
      alert(`Test notification sent to ${channel}`);
    } else {
      alert(`Failed to send test to ${channel}`);
    }
  },

  // First-run wizard
  showWizard() {
    if (localStorage.getItem('sentinel-wizard-done')) return;

    const overlay = Utils.el('div', { className: 'wizard-overlay', id: 'wizard-overlay' });
    overlay.innerHTML = `
      <div class="wizard-card" id="wizard-card">
        <h2>Welcome to SENTINEL</h2>
        <p>Real-time global intelligence monitoring. Let's get you set up in a few quick steps.</p>
        <button class="btn btn--primary" onclick="Settings.wizardStep(1)">Get Started</button>
        <button class="btn" onclick="Settings.dismissWizard()">Skip Setup</button>
      </div>
    `;
    document.body.appendChild(overlay);
  },

  wizardStep(step) {
    const card = document.getElementById('wizard-card');
    if (!card) return;

    switch (step) {
      case 1:
        card.innerHTML = `
          <h2>Your Location</h2>
          <p>Set your home location for proximity alerts and relevant events.</p>
          <button class="btn btn--primary" onclick="Settings.wizardGetLocation()">Use My Location</button>
          <button class="btn" onclick="Settings.wizardStep(2)">Skip</button>
        `;
        break;
      case 2:
        card.innerHTML = `
          <h2>Notifications</h2>
          <p>Enable push notifications to stay informed about critical events.</p>
          <button class="btn btn--primary" onclick="Settings.wizardEnableNotif()">Enable Notifications</button>
          <button class="btn" onclick="Settings.wizardStep(3)">Skip</button>
        `;
        break;
      case 3:
        card.innerHTML = `
          <h2>You're Ready</h2>
          <p>SENTINEL is monitoring global events in real-time. Explore the Signal Board to see what's happening now.</p>
          <button class="btn btn--primary" onclick="Settings.dismissWizard()">Start Monitoring</button>
        `;
        break;
    }
  },

  wizardGetLocation() {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(pos => {
        localStorage.setItem('sentinel-lat', pos.coords.latitude);
        localStorage.setItem('sentinel-lng', pos.coords.longitude);
        this.wizardStep(2);
      }, () => this.wizardStep(2));
    } else {
      this.wizardStep(2);
    }
  },

  wizardEnableNotif() {
    if ('Notification' in window) {
      Notification.requestPermission().then(() => this.wizardStep(3));
    } else {
      this.wizardStep(3);
    }
  },

  dismissWizard() {
    localStorage.setItem('sentinel-wizard-done', '1');
    const overlay = document.getElementById('wizard-overlay');
    if (overlay) overlay.remove();
  }
};
