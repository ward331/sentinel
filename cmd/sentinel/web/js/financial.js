/* ============================================================
   SENTINEL V3 — Financial Overview
   ============================================================ */

const Financial = {
  data: null,
  refreshTimer: null,

  async render(container) {
    container.innerHTML = `
      <div class="section-header">
        <span class="section-header__title">Financial Overview</span>
        <span id="fin-updated" style="font-size:11px;color:var(--text-muted)"></span>
      </div>
      <div id="fin-cards" class="fin-grid"></div>
      <div id="fin-yield" class="yield-card" style="display:none"></div>
      <div id="fin-gauge" class="gauge-container" style="display:none"></div>
    `;

    await this.load();
    this.refreshTimer = setInterval(() => this.load(), 60000);
  },

  async load() {
    const data = await Utils.apiFetch('/api/financial/overview');
    if (data) {
      this.data = data;
      Utils.cacheKV('financial', data);
    } else if (!this.data) {
      this.data = await Utils.getKV('financial');
    }
    this.update();
  },

  update() {
    const d = this.data;
    if (!d) {
      document.getElementById('fin-cards').innerHTML = '<div class="loading">Loading financial data</div>';
      return;
    }

    // Main cards
    const instruments = [
      { key: 'vix', label: 'VIX', prefix: '' },
      { key: 'btc', label: 'BTC', prefix: '$' },
      { key: 'eth', label: 'ETH', prefix: '$' },
      { key: 'oil', label: 'OIL (WTI)', prefix: '$' },
      { key: 'gold', label: 'GOLD', prefix: '$' }
    ];

    const cardsEl = document.getElementById('fin-cards');
    cardsEl.innerHTML = instruments.map(inst => {
      const item = d[inst.key] || d[inst.label.toLowerCase()] || {};
      const value = item.value || item.price || item.last || 0;
      const change = item.change || item.change_pct || 0;
      const changeDir = change >= 0 ? 'up' : 'down';
      const arrow = change >= 0 ? '\u25B2' : '\u25BC';
      const formatted = typeof value === 'number' ? value.toLocaleString(undefined, { maximumFractionDigits: 2 }) : value;

      return `
        <div class="fin-card">
          <div class="fin-card__label">${inst.label}</div>
          <div class="fin-card__value">${inst.prefix}${formatted}</div>
          <div class="fin-card__change fin-card__change--${changeDir}">
            ${arrow} ${Math.abs(change).toFixed(2)}%
          </div>
        </div>
      `;
    }).join('');

    // Yield curve
    const yieldEl = document.getElementById('fin-yield');
    if (d.yield_2y !== undefined || d.treasury_2y !== undefined) {
      const y2 = d.yield_2y || d.treasury_2y || 0;
      const y10 = d.yield_10y || d.treasury_10y || 0;
      const inverted = y2 > y10;
      yieldEl.style.display = 'flex';
      yieldEl.innerHTML = `
        <div class="yield-card__pair">
          <div class="yield-card__rate">${Number(y2).toFixed(2)}%</div>
          <div class="yield-card__label">2Y Treasury</div>
        </div>
        <div class="yield-card__spread">
          <div style="font-size:14px;color:var(--text-muted)">Spread</div>
          <div style="font-size:18px;font-weight:700;color:${inverted ? 'var(--sev-critical)' : 'var(--threat-low)'}">${(y10 - y2).toFixed(2)}%</div>
          ${inverted ? '<span class="badge badge--inverted">INVERTED</span>' : ''}
        </div>
        <div class="yield-card__pair">
          <div class="yield-card__rate">${Number(y10).toFixed(2)}%</div>
          <div class="yield-card__label">10Y Treasury</div>
        </div>
      `;
    }

    // Fear & Greed gauge
    const gaugeEl = document.getElementById('fin-gauge');
    const fng = d.fear_greed || d.fear_and_greed || d.fearGreed;
    if (fng !== undefined && fng !== null) {
      const value = typeof fng === 'object' ? (fng.value || 0) : fng;
      const label = typeof fng === 'object' ? (fng.label || '') : '';
      const rotation = (value / 100) * 180 - 180;
      let gaugeColor = 'var(--sev-critical)';
      if (value > 25) gaugeColor = 'var(--sev-alert)';
      if (value > 45) gaugeColor = 'var(--sev-warning)';
      if (value > 55) gaugeColor = 'var(--threat-low)';
      if (value > 75) gaugeColor = '#059669';

      gaugeEl.style.display = 'flex';
      gaugeEl.innerHTML = `
        <div class="gauge-arc">
          <div class="gauge-arc__bg"></div>
          <div class="gauge-arc__fill" style="border-color:${gaugeColor};transform:rotate(${rotation}deg)"></div>
        </div>
        <div class="gauge-value" style="color:${gaugeColor}">${value}</div>
        <div class="gauge-label">${label || 'Fear & Greed Index'}</div>
      `;
    }

    // Update timestamp
    const updEl = document.getElementById('fin-updated');
    if (updEl) updEl.textContent = 'Updated ' + new Date().toLocaleTimeString();
  },

  destroy() {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = null;
    }
  }
};
