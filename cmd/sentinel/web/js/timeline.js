/* ============================================================
   SENTINEL V3 — Timeline Scrubber
   ============================================================ */

const Timeline = {
  playing: false,
  playTimer: null,
  currentTime: null,

  init() {
    const bar = document.getElementById('timeline-bar');
    if (!bar || bar.dataset.init === '1') return;
    bar.dataset.init = '1';

    const now = Date.now();
    const dayAgo = now - 24 * 60 * 60 * 1000;

    bar.innerHTML = `
      <button class="timeline-bar__play" id="tl-play" onclick="Timeline.togglePlay()" aria-label="Play/Pause">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><polygon points="2,0 12,6 2,12"/></svg>
      </button>
      <input type="range" id="tl-slider" min="${dayAgo}" max="${now}" value="${now}" step="60000">
      <div class="timeline-bar__time" id="tl-time">${new Date(now).toLocaleTimeString()}</div>
    `;

    document.getElementById('tl-slider').addEventListener('input', e => {
      this.currentTime = parseInt(e.target.value);
      this.updateLabel();
      this.filterByTime();
    });
  },

  updateLabel() {
    const label = document.getElementById('tl-time');
    if (label && this.currentTime) {
      label.textContent = new Date(this.currentTime).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
  },

  togglePlay() {
    this.playing = !this.playing;
    const btn = document.getElementById('tl-play');
    if (btn) {
      btn.innerHTML = this.playing
        ? '<svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><rect x="1" y="0" width="4" height="12"/><rect x="7" y="0" width="4" height="12"/></svg>'
        : '<svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><polygon points="2,0 12,6 2,12"/></svg>';
    }

    if (this.playing) {
      this.playTimer = setInterval(() => {
        const slider = document.getElementById('tl-slider');
        if (!slider) { this.stop(); return; }
        let val = parseInt(slider.value) + 5 * 60000; // advance 5 minutes
        if (val > parseInt(slider.max)) {
          val = parseInt(slider.min);
        }
        slider.value = val;
        this.currentTime = val;
        this.updateLabel();
        this.filterByTime();
      }, 500);
    } else {
      this.stop();
    }
  },

  stop() {
    this.playing = false;
    if (this.playTimer) {
      clearInterval(this.playTimer);
      this.playTimer = null;
    }
  },

  filterByTime() {
    if (!MapView.markerLayer || !MapView.events) return;
    const t = this.currentTime;
    const windowMs = 30 * 60000; // 30-minute window

    MapView.markerLayer.clearLayers();
    for (const item of MapView.markers) {
      const evTime = new Date(item.event.timestamp || item.event.created_at).getTime();
      if (evTime <= t && evTime >= t - windowMs) {
        MapView.markerLayer.addLayer(item.marker);
      }
    }
  },

  destroy() {
    this.stop();
    const bar = document.getElementById('timeline-bar');
    if (bar) {
      bar.classList.remove('timeline-bar--active');
      bar.dataset.init = '0';
    }
  }
};
