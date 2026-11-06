/** @typedef {{slideIndex: number, start: number, end: number}} Timing */

/** Sum every visit to a slide; absent slides remain explicit zeroes.
 * @param {Timing[]} first @param {Timing[]} second
 */
export function compareSlides(first, second) {
  /** @type {Map<number, {slide: number, first: number, second: number, delta: number}>} */
  const rows = new Map();
  for (const [side, timings] of /** @type {const} */ ([['first', first], ['second', second]])) {
    for (const timing of timings) {
      const row = rows.get(timing.slideIndex) ?? { slide: timing.slideIndex, first: 0, second: 0, delta: 0 };
      row[side] += timing.end - timing.start;
      rows.set(timing.slideIndex, row);
    }
  }
  return [...rows.values()].sort((a,b) => a.slide-b.slide).map(row => ({ ...row, delta: row.second-row.first }));
}

/** @param {number} requested @param {number} duration */
export function boundedSeek(requested, duration) { return Math.max(0, Math.min(requested, Math.max(0, duration - .001))); }
