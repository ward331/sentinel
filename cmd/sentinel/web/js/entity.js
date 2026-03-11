/* ============================================================
   SENTINEL V3 — Entity Tracker
   ============================================================ */

const Entity = {
  query: '',
  results: null,

  async render(container) {
    container.innerHTML = `
      <div class="section-header">
        <span class="section-header__title">Entity Search</span>
      </div>
      <div class="entity-search-box">
        <input type="text" class="input" id="entity-input" placeholder="Search aircraft, vessels, people, countries..."
               value="${Utils.esc(this.query)}" autocomplete="off">
        <div id="entity-suggestions" class="entity-suggestions"></div>
      </div>
      <div id="entity-results"></div>
    `;

    const input = document.getElementById('entity-input');
    input.addEventListener('input', Utils.debounce(e => {
      this.query = e.target.value;
      if (this.query.length >= 2) {
        this.search(this.query);
      } else {
        document.getElementById('entity-suggestions').classList.remove('entity-suggestions--open');
        document.getElementById('entity-results').innerHTML = '';
      }
    }, 300));

    input.addEventListener('keydown', e => {
      if (e.key === 'Enter' && this.query.length >= 2) {
        document.getElementById('entity-suggestions').classList.remove('entity-suggestions--open');
        this.search(this.query);
      }
    });
  },

  async search(q) {
    const data = await Utils.apiFetch(`/api/entity/search?q=${encodeURIComponent(q)}`);
    this.results = data;
    this.renderResults(data);
  },

  renderResults(data) {
    const container = document.getElementById('entity-results');
    const sugBox = document.getElementById('entity-suggestions');
    sugBox.classList.remove('entity-suggestions--open');

    if (!data) {
      container.innerHTML = '<div class="empty-state"><div class="empty-state__text">No results found</div></div>';
      return;
    }

    const results = Array.isArray(data) ? data : (data.results || data.entities || data.items || []);
    if (results.length === 0 && !data.aircraft && !data.vessels && !data.events) {
      container.innerHTML = '<div class="empty-state"><div class="empty-state__text">No entities match your search</div></div>';
      return;
    }

    let html = '';

    // Group by type if structured
    if (data.aircraft && data.aircraft.length > 0) {
      html += this.renderGroup('Aircraft', data.aircraft, this.renderAircraft);
    }
    if (data.vessels && data.vessels.length > 0) {
      html += this.renderGroup('Vessels', data.vessels, this.renderVessel);
    }
    if (data.people && data.people.length > 0) {
      html += this.renderGroup('People & Organizations', data.people, this.renderPerson);
    }
    if (data.events && data.events.length > 0) {
      html += this.renderGroup('Events', data.events, this.renderEvent);
    }

    // Flat results
    if (!html && results.length > 0) {
      html = this.renderGroup('Results', results, this.renderGeneric);
    }

    container.innerHTML = html || '<div class="empty-state"><div class="empty-state__text">No results</div></div>';
  },

  renderGroup(title, items, renderFn) {
    return `
      <div class="entity-result-group">
        <div class="entity-result-group__header">${title} (${items.length})</div>
        ${items.map(item => renderFn.call(this, item)).join('')}
      </div>
    `;
  },

  renderAircraft(ac) {
    return `
      <div class="card card--clickable">
        <div class="card__header">
          <span class="card__title">${Utils.esc(ac.callsign || ac.icao || ac.registration || 'Unknown')}</span>
          <span class="badge badge--info">${Utils.esc(ac.type || 'Aircraft')}</span>
        </div>
        <div class="card__subtitle">
          ${ac.operator ? `Operator: ${Utils.esc(ac.operator)}<br>` : ''}
          ${ac.altitude ? `Alt: ${ac.altitude}ft` : ''} ${ac.speed ? `Speed: ${ac.speed}kts` : ''} ${ac.heading ? `Hdg: ${ac.heading}&deg;` : ''}
          ${ac.latitude && ac.longitude ? `<br>Pos: ${Number(ac.latitude).toFixed(3)}, ${Number(ac.longitude).toFixed(3)}` : ''}
        </div>
      </div>
    `;
  },

  renderVessel(v) {
    return `
      <div class="card card--clickable">
        <div class="card__header">
          <span class="card__title">${Utils.esc(v.name || v.mmsi || 'Unknown Vessel')}</span>
          <span class="badge badge--watch">${Utils.esc(v.ship_type || v.type || 'Vessel')}</span>
        </div>
        <div class="card__subtitle">
          ${v.destination ? `Dest: ${Utils.esc(v.destination)}<br>` : ''}
          ${v.speed ? `Speed: ${v.speed}kts` : ''}
          ${v.latitude && v.longitude ? `<br>Pos: ${Number(v.latitude).toFixed(3)}, ${Number(v.longitude).toFixed(3)}` : ''}
        </div>
      </div>
    `;
  },

  renderPerson(p) {
    return `
      <div class="card card--clickable">
        <div class="card__header">
          <span class="card__title">${Utils.esc(p.name || 'Unknown')}</span>
          ${p.sanctions ? '<span class="badge badge--critical">SANCTIONED</span>' : ''}
        </div>
        <div class="card__subtitle">
          ${p.mentions ? `${p.mentions} news mentions` : ''}
          ${p.country ? `<br>Country: ${Utils.esc(p.country)}` : ''}
        </div>
      </div>
    `;
  },

  renderEvent(ev) {
    const sev = Utils.severityClass(ev.severity);
    return `
      <div class="card card--clickable" onclick="App.navigate('#/feed?id=${ev.id}')">
        <div class="card__header">
          <span class="card__title">${Utils.esc(ev.title || ev.description || '')}</span>
          <span class="badge badge--${sev}">${Utils.severityLabel(ev.severity)}</span>
        </div>
        <div class="card__subtitle">${Utils.esc(ev.source || '')} &middot; ${Utils.timeAgo(ev.timestamp || ev.created_at)}</div>
      </div>
    `;
  },

  renderGeneric(item) {
    return `
      <div class="card">
        <div class="card__header">
          <span class="card__title">${Utils.esc(item.name || item.title || item.id || 'Item')}</span>
          ${item.type ? `<span class="badge badge--info">${Utils.esc(item.type)}</span>` : ''}
        </div>
        ${item.description ? `<div class="card__subtitle">${Utils.esc(item.description)}</div>` : ''}
      </div>
    `;
  }
};
