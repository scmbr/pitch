// Callers supply a monotonic clock; delayed UI ticks cannot extend a phase.
export class TrainingTimer {
  phase = 'idle';
  paused = false;
  elapsed = 0;
  anchor = 0;
  preparation = 30;
  speech = 120;

  /** @param {number} preparation @param {number} speech @param {number} now */
  start(preparation, speech, now) {
    if (!Number.isFinite(preparation) || preparation < 0 || !Number.isFinite(speech) || speech <= 0) {
      throw new Error('invalidDuration');
    }
    this.preparation = preparation;
    this.speech = speech;
    this.phase = preparation > 0 ? 'preparation' : 'speech';
    this.elapsed = 0;
    this.anchor = now;
    this.paused = false;
  }

  /** @param {number} now */
  tick(now) {
    if (this.phase === 'idle' || this.phase === 'finished' || this.paused) return;
    this.elapsed += Math.max(0, now - this.anchor) / 1000;
    this.anchor = now;
    if (this.phase === 'preparation' && this.elapsed >= this.preparation) {
      this.elapsed -= this.preparation;
      this.phase = 'speech';
    }
  }

  /** @param {number} now */
  togglePause(now) {
    this.tick(now);
    this.paused = !this.paused;
    this.anchor = now;
  }

  /** @param {number} now */
  skip(now) {
    if (this.phase !== 'preparation') return;
    this.phase = 'speech';
    this.elapsed = 0;
    this.anchor = now;
  }

  /** @param {number} now */
  finish(now) { this.tick(now); this.phase = 'finished'; this.paused = false; }

  get remaining() { return (this.phase === 'preparation' ? this.preparation : this.speech) - this.elapsed; }
  get warning() { return this.remaining <= 15; }
  get overtime() { return this.phase !== 'preparation' && this.remaining < 0; }
}

/** @param {number} seconds */
export function formatTime(seconds) {
  const value = Math.ceil(Math.abs(seconds));
  return `${seconds < 0 ? '+' : ''}${Math.floor(value / 60).toString().padStart(2, '0')}:${(value % 60).toString().padStart(2, '0')}`;
}
