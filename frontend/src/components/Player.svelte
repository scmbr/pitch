<script lang="ts">
  import { i18n } from '../i18n/index.svelte';
  import { onDestroy } from 'svelte';
  import type { SlideTiming } from '../../bindings/github.com/scmbr/pitch/internal/attempts/models';
  let { token, duration, versionID = 0, timings = [] }: { token: string; duration: number; versionID?: number; timings?: SlideTiming[] } = $props();
  let video: HTMLVideoElement;
  let speed = $state(1);
  let position = $state(0);
  let currentSlide = $derived(timings.findLast(t => position + .001 >= t.start)?.slideIndex ?? timings[0]?.slideIndex ?? 0);
  const t = i18n.t;
  function seek(event: Event) {
    jump(Number((event.target as HTMLInputElement).value));
  }
  function jump(seconds: number) { position = seconds; video.currentTime = seconds; }
  onDestroy(() => { if (video) { video.pause(); video.removeAttribute('src'); video.load(); } });
</script>

<video bind:this={video} src={'/media/' + token + '.webm'} controls playsinline preload="metadata" ontimeupdate={() => position = video.currentTime} aria-label={t('history.video')}>
  <track kind="captions" />
</video>
<div class="controls">
  <label>{t('history.speed')}
    <select bind:value={speed} onchange={() => video.playbackRate = speed}>
      {#each [.75, 1, 1.25, 1.5, 1.75, 2] as n}<option value={n}>{n}×</option>{/each}
    </select>
  </label>
  <label class="seek">{t('history.seek')}
    <input type="range" min="0" max={duration || 1} step=".1" value={position} oninput={seek} disabled={!duration} />
  </label>
</div>
{#if versionID && timings.length}
  <img class="slide" src={`/slides/${versionID}/${currentSlide + 1}.png`} alt={i18n.t('projects.slideNumber', { page: String(currentSlide + 1) })} />
  <div class="timeline" aria-label={i18n.t('projects.timeline')}>
    {#each timings as timing}
      <button onclick={() => jump(timing.start)}>{i18n.t('projects.slideNumber', { page: String(timing.slideIndex + 1) })} · {timing.start.toFixed(1)}–{timing.end.toFixed(1)}</button>
    {/each}
  </div>
{/if}

<style>
  video { width: 100%; max-height: 55vh; background: #10141d; border-radius: .5rem; }
  .controls { display: flex; gap: 1rem; align-items: center; margin: .75rem 0; }
  label { display: grid; gap: .4rem; } .seek { flex: 1; }
  .slide { width: 100%; max-height: 35vh; object-fit: contain; background: white; }
  .timeline { display: flex; flex-wrap: wrap; gap: .5rem; margin: .75rem 0; }
</style>
