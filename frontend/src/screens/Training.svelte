<script lang="ts">
  import { onDestroy } from 'svelte';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  import { TrainingTimer, formatTime } from '../lib/timer.js';
  import { Recorder } from '../lib/recorder';

  const t = i18n.t;
  const timer = new TrainingTimer();
  let preparation = $state(30);
  let speech = $state(120);
  let phase = $state('idle');
  let remaining = $state(30);
  let paused = $state(false);
  let busy = $state(false);
  let message = $state<MessageKey | null>(null);
  let saved = $state('');
  let video: HTMLVideoElement;
  let recorder: Recorder | null = null;
  let interval: ReturnType<typeof setInterval> | undefined;
  let disposed = false;

  function update() {
    timer.tick(performance.now());
    phase = timer.phase;
    remaining = timer.remaining;
    paused = timer.paused;
    if (phase === 'speech' && recorder && !started) void beginSpeech();
  }
  let started = false;
  let ready = $state(false);
  async function beginSpeech() {
    started = true;
    try {
      await recorder?.start();
      ready = true;
      if (timer.paused) recorder?.pause(true);
    } catch { if (!disposed) await fail('training.diskError'); }
  }

  async function start() {
    if (!Number.isFinite(preparation) || preparation < 0 || preparation > 3600 || !Number.isFinite(speech) || speech < 1 || speech > 86400) {
      message = 'training.durationError'; return;
    }
    busy = true; message = null; saved = ''; started = false; ready = false;
    const capture = new Recorder();
    recorder = capture;
    capture.onFailure = () => { if (!disposed) void fail('training.diskError'); };
    try {
      const stream = await capture.open();
      if (disposed) return;
      video.srcObject = stream;
      await video.play();
      timer.start(preparation, speech, performance.now());
      update();
      interval = setInterval(update, 100);
    } catch {
      if (!disposed) await fail('training.cameraError');
    } finally { busy = false; }
  }

  async function fail(key: MessageKey) {
    clearInterval(interval);
    phase = 'idle';
    message = key;
    await recorder?.abort().catch(() => {});
    recorder = null;
    if (video) video.srcObject = null;
  }

  function pause() {
    timer.togglePause(performance.now());
    recorder?.pause(timer.paused);
    update();
  }

  async function finish() {
    busy = true;
    clearInterval(interval);
    timer.finish(performance.now());
    phase = 'finished';
    try {
      saved = await recorder!.finish();
      recorder = null;
      message = 'training.saved';
    } catch { await fail('training.diskError'); }
    finally { busy = false; if (video) video.srcObject = null; }
  }

  async function abort() {
    busy = true;
    await fail('training.discarded');
    busy = false;
  }

  onDestroy(() => {
    disposed = true;
    clearInterval(interval);
    recorder?.release();
    void recorder?.abort().catch(() => {});
  });
</script>

<main>
  <h1>{t('training.title')}</h1>
  <div class="durations">
    <label>{t('training.preparation')} <input type="number" min="0" max="3600" bind:value={preparation} disabled={phase !== 'idle' && phase !== 'finished'} /></label>
    <label>{t('training.speech')} <input type="number" min="1" max="86400" bind:value={speech} disabled={phase !== 'idle' && phase !== 'finished'} /></label>
  </div>
  <div class="stage">
    <video bind:this={video} autoplay muted playsinline aria-label={t('training.preview')}></video>
    <div class="clock">
      <p>{t(phase === 'preparation' ? 'training.preparing' : 'training.speaking')}</p>
      <output class:warning={remaining <= 15} class:overtime={remaining < 0}>{formatTime(remaining)}</output>
      {#if phase === 'speech'}<p class="recording" role="status">● {t(paused ? 'training.paused' : 'training.recording')}</p>{/if}
    </div>
  </div>
  <div class="actions">
    {#if phase === 'idle' || phase === 'finished'}
      <button class="primary" onclick={start} disabled={busy}>{t('training.start')}</button>
    {:else}
      <button onclick={pause} disabled={busy}>{t(paused ? 'training.resume' : 'training.pause')}</button>
      {#if phase === 'preparation'}
        <button onclick={() => { timer.skip(performance.now()); update(); }} disabled={busy}>{t('training.skip')}</button>
      {:else}
        <button class="primary" onclick={finish} disabled={busy || !ready}>{t('training.finish')}</button>
      {/if}
      <button onclick={abort} disabled={busy}>{t('training.abort')}</button>
    {/if}
  </div>
  {#if message}<p role={message.includes('Error') ? 'alert' : 'status'}>{t(message)}</p>{/if}
  {#if saved}<p class="saved">{t('training.savedFile', { name: saved + '.webm' })}</p>{/if}
</main>

<style>
  main { max-width: 64rem; margin: auto; padding: 2rem 1.5rem; }
  .durations, .actions { display: flex; gap: 1rem; flex-wrap: wrap; margin: 1rem 0; }
  label { display: grid; gap: .4rem; }
  input { width: 10rem; }
  .stage { display: grid; grid-template-columns: 1fr 1fr; align-items: center; gap: 2rem; }
  video { width: 100%; aspect-ratio: 16 / 9; background: #151923; border-radius: .75rem; transform: scaleX(-1); }
  .clock { text-align: center; }
  output { font-size: clamp(3rem, 8vw, 6rem); font-variant-numeric: tabular-nums; }
  .warning, .recording { color: var(--danger); }
  .overtime { text-decoration: underline; }
  .saved { overflow-wrap: anywhere; color: var(--muted); }
  @media(max-width: 650px) { .stage { grid-template-columns: 1fr; } }
</style>
