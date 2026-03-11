/* ============================================================
   SENTINEL V3 — Core App (Router, SSE, Init)
   ============================================================ */

const App = {
  currentView: null,
  sse: null,
  sseRetryMs: 1000,
  sseMaxRetry: 30000,

  // ── Router ──────────────────────────────────────────────────
  routes: {
    '#/signal-board': { name: 'Signal Board', render: () => SignalBoard.render(App.viewContainer()), onSSE: e => SignalBoard.onSSE(e) },
    '#/feed':         { name: 'Feed',         render: () => Feed.render(App.viewContainer()),        onSSE: e => Feed.onSSE(e) },
    '#/map':          { name: 'Map',          render: () => MapView.render(App.viewContainer()),     onSSE: e => MapView.onSSE(e) },
    '#/financial':    { name: 'Financial',    render: () => Financial.render(App.viewContainer()) },
    '#/entity':       { name: 'Entity',       render: () => Entity.render(App.viewContainer()) },
    '#/settings':     { name: 'Settings',     render: () => Settings.render(App.viewContainer()) },
  },

  viewContainer() {
    return document.getElementById('view-container');
  },

  navigate(hash) {
    // Clean up previous view
    if (this.currentView === '#/map') {
      MapView.destroy();
      Timeline.destroy();
    }
    if (this.currentView === '#/financial') {
      Financial.destroy();
    }
    if (Feed.detailOpen) Feed.closeDetail();

    const routeKey = hash.split('?')[0];
    location.hash = hash;
    this.currentView = routeKey;
    this.updateNav(routeKey);

    const route = this.routes[routeKey];
    if (route) {
      route.render();
    } else {
      this.navigate('#/signal-board');
    }
  },

  updateNav(routeKey) {
    document.querySelectorAll('.bottom-nav__item').forEach(el => {
      el.classList.toggle('bottom-nav__item--active', el.dataset.route === routeKey);
    });
    document.querySelectorAll('.sidebar__item').forEach(el => {
      el.classList.toggle('sidebar__item--active', el.dataset.route === routeKey);
    });
  },

  // ── SSE ─────────────────────────────────────────────────────
  connectSSE() {
    if (this.sse) {
      this.sse.close();
    }

    this.sse = new EventSource('/api/events/stream');
    this.setStatus('connecting');

    this.sse.onopen = () => {
      this.setStatus('connected');
      this.sseRetryMs = 1000;
    };

    this.sse.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.handleSSE(data);
      } catch (e) {}
    };

    this.sse.addEventListener('event', (e) => {
      try {
        const data = JSON.parse(e.data);
        this.handleSSE({ type: 'new_event', data });
      } catch (err) {}
    });

    this.sse.addEventListener('signal_board', (e) => {
      try {
        const data = JSON.parse(e.data);
        this.handleSSE({ type: 'signal_board', data });
      } catch (err) {}
    });

    this.sse.onerror = () => {
      this.setStatus('disconnected');
      this.sse.close();
      setTimeout(() => this.connectSSE(), this.sseRetryMs);
      this.sseRetryMs = Math.min(this.sseRetryMs * 2, this.sseMaxRetry);
    };
  },

  handleSSE(data) {
    const route = this.routes[this.currentView];
    if (route && route.onSSE) {
      route.onSSE(data);
    }
    if (data.type === 'signal_board') {
      SignalBoard.onSSE(data);
    }
  },

  setStatus(status) {
    const dot = document.getElementById('status-dot');
    const label = document.getElementById('status-label');
    if (dot) {
      dot.className = 'status-dot';
      if (status === 'disconnected') dot.classList.add('status-dot--disconnected');
      if (status === 'connecting') dot.classList.add('status-dot--connecting');
    }
    if (label) {
      const labels = { connected: 'LIVE', disconnected: 'OFFLINE', connecting: 'CONNECTING' };
      label.textContent = labels[status] || status;
    }
  },

  // ── Brief Me ────────────────────────────────────────────────
  async briefMe() {
    const modal = document.getElementById('briefme-modal');
    modal.classList.add('briefme-modal--open');

    const content = document.getElementById('briefme-content');
    content.innerHTML = '<div class="loading">Generating briefing</div>';

    const data = await Utils.apiFetch('/api/intel/briefing');
    if (data) {
      const briefing = data.briefing || data.summary || data.text || JSON.stringify(data, null, 2);
      content.innerHTML = `
        <div style="font-size:13px;line-height:1.6;color:var(--text-secondary);white-space:pre-wrap">${Utils.esc(briefing)}</div>
      `;
    } else {
      content.innerHTML = '<div class="empty-state"><div class="empty-state__text">Briefing unavailable</div></div>';
    }
  },

  closeBriefMe() {
    document.getElementById('briefme-modal').classList.remove('briefme-modal--open');
  },

  // ── Init ────────────────────────────────────────────────────
  init() {
    window.addEventListener('hashchange', () => {
      this.navigate(location.hash || '#/signal-board');
    });

    this.connectSSE();
    Settings.showWizard();

    const hash = location.hash || '#/signal-board';
    this.navigate(hash);
  }
};

document.addEventListener('DOMContentLoaded', () => App.init());
