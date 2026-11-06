<script lang="ts">
  import { onDestroy } from 'svelte';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  import { TrainingTimer, formatTime } from '../lib/timer.js';
  import { Recorder } from '../lib/recorder';
  import type { Version } from '../../bindings/github.com/scmbr/pitch/internal/projects/models';
  import type { SlideTiming } from '../../bindings/github.com/scmbr/pitch/internal/attempts/models';
  let { projectVersion = null, projectName = '', clearProject = () => {} }: { projectVersion?: Version | null; projectName?: string; clearProject?: () => void } = $props();
  let slide = $state(0);
  let timings: SlideTiming[] = [];
  let slideStart = 0;

  const t = i18n.t;
  const timer = new TrainingTimer();
  let preparation = $state(30);
  let speech = $state(120);
  let phase = $state('idle');
  let remaining = $state(30);
  let displayedTime = $derived(phase === 'idle' ? (preparation || speech) : remaining);
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
    timings = []; slideStart = 0;
    try {
      await recorder?.start();
      ready = true;
      if (timer.paused) recorder?.pause(true);
    } catch { if (!disposed) await fail('training.diskError'); }
  }
  function changeSlide(next: number) {
    if (!projectVersion || next < 0 || next >= projectVersion.slideCount || next === slide || busy) return;
    timer.tick(performance.now());
    if (phase === 'speech' && timer.elapsed > slideStart) {
      timings.push({ slideIndex: slide, start: slideStart, end: timer.elapsed });
      slideStart = timer.elapsed;
    }
    slide = next;
  }
  function keys(event: KeyboardEvent) {
    if (event.target instanceof Element && event.target.closest('input,textarea,select,[contenteditable]')) return;
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault(); changeSlide(slide + (event.key === 'ArrowRight' ? 1 : -1));
    }
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
    if (projectVersion && timer.elapsed > slideStart) timings.push({ slideIndex: slide, start: slideStart, end: timer.elapsed });
    phase = 'finished';
    try {
      let thumbnail = '';
      if (video.videoWidth && video.readyState >= 2) {
        const canvas = document.createElement('canvas');
        canvas.width = 320; canvas.height = 180;
        canvas.getContext('2d')?.drawImage(video, 0, 0, 320, 180);
        thumbnail = canvas.toDataURL('image/jpeg', .75).split(',')[1];
      }
      saved = await recorder!.finish({ duration: timer.elapsed, thumbnail, projectId: projectVersion?.projectId ?? 0, versionId: projectVersion?.id ?? 0, topic: '', timings: projectVersion ? timings : [] });
      recorder = null;
      message = 'training.saved';
    } catch (error) { await fail(String(error).includes('storageLimit') ? 'history.limitError' : 'training.diskError'); }
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

<svelte:window onkeydown={keys} />

<main>
  <h1>{t('training.title')}</h1>
  {#if projectVersion}
    <p>{projectName} · {t('projects.versionNumber', { number: String(projectVersion.number) })}</p>
    {#if phase === 'idle' || phase === 'finished'}<button onclick={clearProject}>{t('history.free')}</button>{/if}
    <img class="slide" src={`/slides/${projectVersion.id}/${slide + 1}.png`} alt={t('projects.slide', { page: String(slide + 1), count: String(projectVersion.slideCount) })} />
    <div class="actions">
      <button onclick={() => changeSlide(slide - 1)} disabled={slide === 0 || busy}>{t('projects.previous')}</button>
      <span>{slide + 1} / {projectVersion.slideCount}</span>
      <button onclick={() => changeSlide(slide + 1)} disabled={slide + 1 >= projectVersion.slideCount || busy}>{t('projects.next')}</button>
    </div>
  {/if}
  <div class="durations">
    <label>{t('training.preparation')} <input type="number" min="0" max="3600" bind:value={preparation} disabled={phase !== 'idle' && phase !== 'finished'} /></label>
    <label>{t('training.speech')} <input type="number" min="1" max="86400" bind:value={speech} disabled={phase !== 'idle' && phase !== 'finished'} /></label>
  </div>
  <div class="stage">
    <video bind:this={video} autoplay muted playsinline aria-label={t('training.preview')}></video>
    <div class="clock">
      <p>{t(phase === 'idle' ? 'training.ready' : phase === 'preparation' ? 'training.preparing' : 'training.speaking')}</p>
      <output class:warning={displayedTime <= 15} class:overtime={displayedTime < 0}>{formatTime(displayedTime)}</output>
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
  .slide { display: block; width: 100%; max-height: 45vh; object-fit: contain; background: white; }
  @media(max-width: 650px) { .stage { grid-template-columns: 1fr; } }
</style>
