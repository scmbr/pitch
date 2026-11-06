<script lang="ts">
  import { onDestroy } from 'svelte';
  import type { Attempt, SlideTiming } from '../../bindings/github.com/scmbr/pitch/internal/attempts/models';
  import { AttemptService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import { i18n } from '../i18n/index.svelte';
  import { compareSlides, boundedSeek } from '../lib/comparison.js';
  import { formatTime } from '../lib/timer.js';
  let { first, second, close }: { first: Attempt; second: Attempt; close: () => void } = $props();
  const t = i18n.t;
  let left: HTMLVideoElement;
  let right: HTMLVideoElement;
  let playing = $state(false);
  let position = $state(0);
  let speed = $state(1);
  let audio = $state('left');
  let firstTimings = $state<SlideTiming[]>([]);
  let secondTimings = $state<SlideTiming[]>([]);
  let rows = $derived(compareSlides(firstTimings, secondTimings));
  let duration = $derived(Math.max(first.duration, second.duration));
  let message = $state('');
  let ready = $state(false);
  let frame = 0;
  let generation = 0;
  let startingGeneration = -1;
  let disposed = false;

  $effect(() => {
    void Promise.all([AttemptService.Timings(first.id),AttemptService.Timings(second.id)])
      .then(([a,b]) => { if (!disposed) { firstTimings = a ?? []; secondTimings = b ?? []; } })
      .catch(() => { message = t('history.error'); });
  });
  function loaded() { ready = left?.readyState >= 2 && right?.readyState >= 2; }
  function pause() { generation++; playing = false; cancelAnimationFrame(frame); left?.pause(); right?.pause(); }
  function tick() {
    if (!playing || disposed) return;
    const master = !left.ended ? left : right;
    position = master.currentTime;
    const follower = master === left ? right : left;
    const followerDuration = master === left ? second.duration : first.duration;
    if (!follower.ended && position < followerDuration && Math.abs(follower.currentTime-position) > .15) follower.currentTime = position;
    if (left.ended && right.ended) { pause(); position = duration; return; }
    frame = requestAnimationFrame(tick);
  }
  async function play() {
    if (position >= duration-.01) seek(0);
    message = ''; const current = ++generation; playing = true;
    startingGeneration = current;
    try {
      await Promise.all([left,right].map(v => v.ended && position > .001 ? Promise.resolve() : v.play()));
      if (current !== generation || disposed) return;
      startingGeneration = -1;
      frame = requestAnimationFrame(tick);
    } catch { if (current === generation) { pause(); message = t('compare.playError'); } }
  }
  function seek(seconds: number) {
    const resume = playing;
    pause();
    position = Math.max(0,Math.min(seconds,duration));
    left.currentTime = boundedSeek(position,Number.isFinite(left.duration) ? left.duration : first.duration);
    right.currentTime = boundedSeek(position,Number.isFinite(right.duration) ? right.duration : second.duration);
    if (resume && position < duration-.01) void play();
  }
  function buffering() { if (playing && startingGeneration !== generation) { pause(); message = t('compare.buffering'); } }
  function failed() { pause(); message = t('compare.playError'); }
  onDestroy(() => {
    disposed = true; pause();
    for (const v of [left,right]) { if (v) { v.removeAttribute('src'); v.load(); } }
  });
</script>

<section class="comparison" aria-label={t('compare.title')}>
  <div class="heading"><h2>{t('compare.title')}</h2><button onclick={close}>{t('compare.back')}</button></div>
  {#if first.versionId !== second.versionId}<p>{t('compare.versionsWarning')}</p>{/if}
  <div class="videos">
    <figure><figcaption>{t('compare.first')} · {new Date(first.createdAt).toLocaleString(i18n.language)}{#if first.versionNumber} · {t('projects.versionNumber',{number:String(first.versionNumber)})}{/if}</figcaption>
      <video bind:this={left} src={'/media/'+first.token+'.webm'} playsinline preload="auto" muted={audio !== 'left'} onloadeddata={loaded} onwaiting={buffering} onerror={failed} aria-label={t('compare.first')}><track kind="captions" /></video>
    </figure>
    <figure><figcaption>{t('compare.second')} · {new Date(second.createdAt).toLocaleString(i18n.language)}{#if second.versionNumber} · {t('projects.versionNumber',{number:String(second.versionNumber)})}{/if}</figcaption>
      <video bind:this={right} src={'/media/'+second.token+'.webm'} playsinline preload="auto" muted={audio !== 'right'} onloadeddata={loaded} onwaiting={buffering} onerror={failed} aria-label={t('compare.second')}><track kind="captions" /></video>
    </figure>
  </div>
  <div class="controls">
    <button class="primary" onclick={() => playing ? pause() : play()} disabled={!ready}>{t(playing ? 'training.pause' : 'compare.play')}</button>
    <label class="seek">{formatTime(position)} / {formatTime(duration)}
      <input type="range" min="0" max={duration || 1} step=".1" value={position} disabled={!ready} oninput={e => seek(Number(e.currentTarget.value))} aria-label={t('history.seek')} />
    </label>
    <label>{t('history.speed')}<select bind:value={speed} onchange={() => { left.playbackRate=speed; right.playbackRate=speed; }}>{#each [.75,1,1.25,1.5,1.75,2] as value}<option {value}>{value}×</option>{/each}</select></label>
    <label>{t('compare.audio')}<select bind:value={audio}><option value="left">{t('compare.first')}</option><option value="right">{t('compare.second')}</option><option value="none">{t('compare.muted')}</option></select></label>
  </div>
  {#if message}<p role="alert">{message}</p>{/if}
  {#if rows.length}
    <table><caption>{t('compare.slideTimes')}</caption><thead><tr><th>{t('compare.slide')}</th><th>{t('compare.first')}</th><th>{t('compare.second')}</th><th>{t('compare.delta')}</th></tr></thead>
      <tbody>{#each rows as row}<tr><th scope="row">{row.slide+1}</th><td>{row.first.toFixed(1)}</td><td>{row.second.toFixed(1)}</td><td>{row.delta>0?'+':''}{row.delta.toFixed(1)}</td></tr>{/each}</tbody>
    </table>
  {:else}<p>{t('compare.noTimeline')}</p>{/if}
</section>

<style>
  .heading, .controls { display:flex; align-items:center; gap:1rem; flex-wrap:wrap; margin:1rem 0; }
  .heading h2 { margin-right:auto; }
  .videos { display:grid; grid-template-columns:1fr 1fr; gap:1rem; }
  figure { margin:0; min-width:0; } figcaption { margin-bottom:.5rem; font-size:.9rem; }
  video { width:100%; aspect-ratio:16/9; background:#10141d; border-radius:.5rem; }
  label { display:grid; gap:.4rem; } .seek { flex:1; min-width:10rem; }
  table { width:100%; border-collapse:collapse; margin-top:2rem; font-variant-numeric:tabular-nums; }
  th,td { text-align:left; border-bottom:1px solid var(--border); padding:.75rem; }
  caption { text-align:left; font-weight:600; margin-bottom:.5rem; }
  @media(max-width:650px) { .videos { grid-template-columns:1fr; } }
</style>
