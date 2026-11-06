import { test } from 'node:test';
import assert from 'node:assert/strict';
import { TrainingTimer, formatTime } from '../src/lib/timer.js';

test('delayed tick carries elapsed preparation into speech and overtime', () => {
  const t = new TrainingTimer();
  t.start(30, 60, 1000);
  t.tick(41000);
  assert.equal(t.phase, 'speech');
  assert.equal(t.elapsed, 10);
  t.tick(96000);
  assert.equal(t.remaining, -5);
  assert.equal(t.overtime, true);
});
test('pause excludes paused time, skip resets preparation, finish freezes timer', () => {
  const t = new TrainingTimer();
  t.start(30, 60, 0);
  t.togglePause(5000);
  t.tick(100000);
  assert.equal(t.elapsed, 5);
  t.togglePause(105000);
  t.skip(106000);
  t.tick(151000);
  assert.equal(t.remaining, 15);
  assert.equal(t.warning, true);
  t.finish(152000);
  t.tick(200000);
  assert.equal(t.elapsed, 46);
});
test('zero preparation and invalid durations', () => {
  const t = new TrainingTimer();
  t.start(0, 1, 0);
  assert.equal(t.phase, 'speech');
  assert.throws(() => t.start(-1, 1, 0));
  assert.throws(() => t.start(0, NaN, 0));
  assert.equal(formatTime(-65), '+01:05');
});
