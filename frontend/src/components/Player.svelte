<script lang="ts">
  import { i18n } from '../i18n/index.svelte';
  import { onDestroy } from 'svelte';
  let { token, duration }: { token: string; duration: number } = $props();
  let video: HTMLVideoElement;
  let speed = $state(1);
  let position = $state(0);
  const t = i18n.t;
  function seek(event: Event) {
    video.currentTime = Number((event.target as HTMLInputElement).value);
  }
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

<style>
  video { width: 100%; max-height: 55vh; background: #10141d; border-radius: .5rem; }
  .controls { display: flex; gap: 1rem; align-items: center; margin: .75rem 0; }
  label { display: grid; gap: .4rem; } .seek { flex: 1; }
</style>
