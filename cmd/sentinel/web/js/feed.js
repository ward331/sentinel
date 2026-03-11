/* ============================================================
   SENTINEL V3 — Event Feed
   ============================================================ */

const Feed = {
  events: [],
  offset: 0,
  limit: 30,
  hasMore: true,
  filters: { category: '', severity: '', truthMin: 0, search: '' },
  detailOpen: false,

  async render(container) {
    // Parse query params from hash
    const hash = location.hash;
    const qIdx = hash.indexOf('?');
    if (qIdx > -1) {
      const params = new URLSearchParams(hash.slice(qIdx + 1));
      if (params.get('category')) this.filters.category = params.get('category');
      if (params.get('id')) {
        // Show specific event detail after loading
        setTimeout(() => this.showDetail(params.get('id')), 300);
      }
    }

    container.innerHTML = `
      <div class="section-header">
        <span class="section-header__title">Event Feed</span>
      </div>
      <div class="filter-bar">
        <input type="text" class="input" id="feed-search" placeholder="Search events..." value="${Utils.esc(this.filters.search)}">
        <select class="input" id="feed-cat" style="max-width:130px">
          <option value="">All Types</option>
          <option value="earthquake">Earthquake</option>
          <option value="weather">Weather</option>
          <option value="military">Military</option>
          <option value="cyber">Cyber</option>
          <option value="financial">Financial</option>
          <option value="health">Health</option>
          <option value="aviation">Aviation</option>
          <option value="maritime">Maritime</option>
          <option value="space">Space</option>
          <option value="fire">Wildfire</option>
        </select>
        <select class="input" id="feed-sev" style="max-width:120px">
          <option value="">All Severity</option>
          <option value="critical">Critical</option>
          <option value="alert">Alert</option>
          <option value="warning">Warning</option>
          <option value="watch">Watch</option>
          <option value="info">Info</option>
        </select>
      </div>
      <div id="feed-list"></div>
      <div id="feed-load-more" class="load-more" style="display:none">
        <button class="btn btn--small" onclick="Feed.loadMore()">Load More</button>
      </div>
      <div id="event-detail" class="detail-panel" role="dialog" aria-label="Event Detail">
        <button class="detail-panel__close" onclick="Feed.closeDetail()" aria-label="Close">&times;</button>
        <div id="event-detail-content"></div>
      </div>
    `;

    // Bind filters
    document.getElementById('feed-search').addEventListener('input', Utils.debounce(e => {
      this.filters.search = e.target.value;
      this.reload();
    }, 300));

    document.getElementById('feed-cat').addEventListener('change', e => {
      this.filters.category = e.target.value;
      this.reload();
    });

    document.getElementById('feed-sev').addEventListener('change', e => {
      this.filters.severity = e.target.value;
      this.reload();
    });

    // Set filter values
    if (this.filters.category) document.getElementById('feed-cat').value = this.filters.category;
    if (this.filters.severity) document.getElementById('feed-sev').value = this.filters.severity;

    this.events = [];
    this.offset = 0;
    this.hasMore = true;
    await this.loadMore();
  },

  buildQuery() {
    const params = new URLSearchParams();
    params.set('limit', this.limit);
    params.set('offset', this.offset);
    if (this.filters.category) params.set('category', this.filters.category);
    if (this.filters.severity) params.set('severity', this.filters.severity);
    if (this.filters.search) params.set('q', this.filters.search);
    return params.toString();
  },

  async loadMore() {
    const data = await Utils.apiFetch(`/api/events?${this.buildQuery()}`);
    const list = document.getElementById('feed-list');
    const loadMoreBtn = document.getElementById('feed-load-more');

    let newEvents = [];
    if (data) {
      newEvents = Array.isArray(data) ? data : (data.events || data.items || []);
      Utils.cacheEvents(newEvents);
    } else if (this.events.length === 0) {
      newEvents = await Utils.getCachedEvents();
    }

    if (newEvents.length < this.limit) this.hasMore = false;
    this.events = this.events.concat(newEvents);
    this.offset += newEvents.length;

    // Render new events
    for (const ev of newEvents) {
      list.appendChild(this.renderEventCard(ev));
    }

    if (this.events.length === 0) {
      list.innerHTML = '<div class="empty-state"><div class="empty-state__text">No events found</div></div>';
    }

    loadMoreBtn.style.display = this.hasMore ? 'flex' : 'none';
  },

  reload() {
    this.events = [];
    this.offset = 0;
    this.hasMore = true;
    const list = document.getElementById('feed-list');
    if (list) list.innerHTML = '';
    this.loadMore();
  },

  renderEventCard(ev) {
    const sev = Utils.severityClass(ev.severity);
    const truthScore = ev.truth_score || ev.truthScore || 0;
    const card = Utils.el('div', {
      className: 'event-card',
      onClick: () => this.showDetail(ev.id, ev)
    }, [
      Utils.el('div', { className: `event-card__severity event-card__severity--${sev}` }),
      Utils.el('div', { className: 'event-card__body' }, [
        Utils.el('div', { className: 'event-card__title', textContent: ev.title || ev.description || 'Untitled' }),
        Utils.el('div', { className: 'event-card__meta' }, [
          Utils.el('span', { className: 'event-card__source', textContent: ev.source || '' }),
          Utils.el('span', { textContent: ev.location || '' }),
          Utils.el('span', { textContent: Utils.timeAgo(ev.timestamp || ev.created_at) })
        ])
      ]),
      Utils.el('div', { className: 'event-card__right' }, [
        Utils.el('span', { className: `badge badge--${sev}`, textContent: Utils.severityLabel(ev.severity) }),
        truthScore > 0 ? Utils.el('span', {
          className: `truth-score truth-score--${truthScore}`,
          textContent: Utils.truthScoreSymbol(truthScore),
          title: `Truth Score: ${truthScore}/5`
        }) : null
      ].filter(Boolean))
    ]);
    return card;
  },

  async showDetail(id, cachedEvent) {
    let ev = cachedEvent;
    if (!ev || (id && (!ev || ev.id !== id))) {
      ev = await Utils.apiFetch(`/api/events/${id}`);
    }
    if (!ev) return;

    const content = document.getElementById('event-detail-content');
    const sev = Utils.severityClass(ev.severity);
    const ts = ev.truth_score || ev.truthScore || 0;

    content.innerHTML = `
      <div class="detail-panel__title">${Utils.esc(ev.title || ev.description || 'Event')}</div>
      <div style="display:flex;gap:8px;margin-bottom:16px;flex-wrap:wrap">
        <span class="badge badge--${sev}">${Utils.severityLabel(ev.severity)}</span>
        ${ts > 0 ? `<span class="truth-score truth-score--${ts}" title="Truth Score">${Utils.truthScoreSymbol(ts)}</span>` : ''}
        ${ev.source ? `<span class="badge" style="background:var(--bg-card);color:var(--text-secondary)">${Utils.esc(ev.source)}</span>` : ''}
      </div>
      <div class="detail-panel__section">
        <div class="detail-panel__label">Description</div>
        <div class="detail-panel__text">${Utils.esc(ev.description || ev.title || 'No description available')}</div>
      </div>
      ${ev.location ? `
        <div class="detail-panel__section">
          <div class="detail-panel__label">Location</div>
          <div class="detail-panel__text">${Utils.esc(ev.location)}</div>
        </div>
      ` : ''}
      ${ev.latitude && ev.longitude ? `
        <div class="detail-panel__section">
          <div class="detail-panel__label">Coordinates</div>
          <div class="detail-panel__text">${ev.latitude.toFixed(4)}, ${ev.longitude.toFixed(4)}</div>
        </div>
      ` : ''}
      <div class="detail-panel__section">
        <div class="detail-panel__label">Time</div>
        <div class="detail-panel__text">${Utils.formatDateTime(ev.timestamp || ev.created_at)}</div>
      </div>
      ${ev.category ? `
        <div class="detail-panel__section">
          <div class="detail-panel__label">Category</div>
          <div class="detail-panel__text">${Utils.esc(ev.category)}</div>
        </div>
      ` : ''}
      <div style="margin-top:20px">
        <button class="btn btn--primary" onclick="Feed.acknowledgeEvent('${ev.id}')">Acknowledge</button>
      </div>
    `;

    document.getElementById('event-detail').classList.add('detail-panel--open');
    this.detailOpen = true;
  },

  closeDetail() {
    document.getElementById('event-detail').classList.remove('detail-panel--open');
    this.detailOpen = false;
  },

  async acknowledgeEvent(id) {
    await Utils.apiFetch(`/api/events/${id}/acknowledge`, { method: 'POST' });
    this.closeDetail();
  },

  onSSE(eventData) {
    if (eventData && (eventData.type === 'new_event' || eventData.type === 'event')) {
      const ev = eventData.data || eventData;
      if (!ev.id) return;
      // Check if it passes current filters
      if (this.filters.severity && ev.severity !== this.filters.severity) return;
      if (this.filters.category && ev.category !== this.filters.category) return;

      this.events.unshift(ev);
      const list = document.getElementById('feed-list');
      if (list) {
        const card = this.renderEventCard(ev);
        list.insertBefore(card, list.firstChild);
      }
    }
  }
};
