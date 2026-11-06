<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { AttemptService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import type { Attempt } from '../../bindings/github.com/scmbr/pitch/internal/attempts/models';
  import Player from '../components/Player.svelte';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  import { formatTime } from '../lib/timer.js';
  const t = i18n.t;
  let attempts = $state<Attempt[]>([]);
  let selected = $state<Attempt | null>(null);
  let scores = $state({ structure: 3, confidence: 3, timing: 3 });
  let note = $state('');
  let busy = $state(false);
  let message = $state<MessageKey | null>(null);
  let limit = $state(0);
  let autoDelete = $state(false);
  let confirmDelete = $state(false);

  async function load() {
    attempts = (await AttemptService.List()) ?? [];
    const policy = await AttemptService.GetPolicy();
    limit = policy.limitMB; autoDelete = policy.autoDelete;
  }
  onMount(() => { void load().catch(() => message = 'history.error'); });
  function choose(a: Attempt) {
    selected = a; confirmDelete = false; message = null;
    scores = { structure: a.scores.structure || 3, confidence: a.scores.confidence || 3, timing: a.scores.timing || 3 };
    note = a.note;
  }
  async function action(task: () => Promise<unknown>) {
    busy = true; message = null;
    try { await task(); await load(); message = 'history.saved'; }
    catch { message = 'history.error'; } finally { busy = false; }
  }
  async function save() {
    if (!selected) return;
    await action(() => AttemptService.Update(selected!.id, scores, note));
  }
  async function remove() {
    if (!selected) return;
    const id = selected.id;
    // Unmount the player before deleting a video that the WebView may hold open.
    selected = null;
    await tick();
    await action(() => AttemptService.Delete(id));
  }
  async function policy(event: SubmitEvent) {
    event.preventDefault();
    await action(() => AttemptService.SetPolicy({ limitMB: limit, autoDelete }));
  }
</script>

<main>
  <h1>{t('history.title')}</h1>
  <form class="policy" onsubmit={policy}>
    <label>{t('history.limit')} <input type="number" min="0" max="1048576" bind:value={limit} required /></label>
    <label class="check"><input type="checkbox" bind:checked={autoDelete} /> {t('history.autoDelete')}</label>
    <button disabled={busy}>{t('settings.save')}</button>
    <p>{t('history.limitHint')}</p>
  </form>
  {#if message}<p role={message === 'history.error' ? 'alert' : 'status'}>{t(message)}</p>{/if}
  <div class="layout">
    <section class="list" aria-label={t('history.title')}>
      {#if !attempts.length}<p>{t('history.empty')}</p>{/if}
      {#each attempts as a (a.id)}
        <button class="attempt" class:active={selected?.id === a.id} onclick={() => choose(a)}>
          {#if a.thumbnail}<img src={'/media/' + a.token + '.jpg'} alt="" loading="lazy" />{/if}
          <span><strong>{a.topic || t('history.free')}</strong><br />{new Date(a.createdAt).toLocaleString(i18n.language)} · {formatTime(a.duration)}
          {#if a.scores.structure}<br />{t('history.ratings', { structure: String(a.scores.structure), confidence: String(a.scores.confidence), timing: String(a.scores.timing) })}{/if}
          {#if a.deleting}<br />{t('history.deleting')}{/if}</span>
        </button>
      {/each}
    </section>
    {#if selected}
      <section class="detail" aria-label={t('history.detail')}>
        {#key selected.id}<Player token={selected.token} duration={selected.duration} />{/key}
        <form onsubmit={event => { event.preventDefault(); void save(); }}>
          <div class="ratings">
            {#each ['structure', 'confidence', 'timing'] as key}
              <label>{t(('history.' + key) as MessageKey)}
                <select bind:value={scores[key as keyof typeof scores]}>{#each [1,2,3,4,5] as n}<option value={n}>{n}</option>{/each}</select>
              </label>
            {/each}
          </div>
          <label>{t('history.note')} <textarea bind:value={note} maxlength="5000" rows="4"></textarea></label>
          <div class="actions"><button class="primary" disabled={busy}>{t('settings.save')}</button>
            <button type="button" onclick={() => confirmDelete = true} disabled={busy}>{t('history.delete')}</button>
          </div>
        </form>
        {#if confirmDelete}
          <div class="confirm" role="alert"><p>{t('history.confirmDelete')}</p>
            <button onclick={remove} disabled={busy}>{t('history.delete')}</button>
            <button onclick={() => confirmDelete = false}>{t('history.cancel')}</button>
          </div>
        {/if}
      </section>
    {/if}
  </div>
</main>

<style>
  main { max-width: 75rem; margin: auto; padding: 2rem 1.5rem; }
  .policy, .ratings, .actions { display: flex; flex-wrap: wrap; align-items: end; gap: 1rem; }
  .policy { border-bottom: 1px solid var(--border); padding-bottom: 1rem; margin-bottom: 1.5rem; }
  .policy p { width: 100%; margin: 0; color: var(--muted); font-size: .85rem; }
  label { display: grid; gap: .4rem; margin-bottom: .75rem; }
  label.check { display: flex; align-items: center; }
  .layout { display: grid; grid-template-columns: minmax(16rem, 1fr) minmax(20rem, 2fr); gap: 2rem; }
  .list { display: flex; flex-direction: column; gap: .75rem; }
  .attempt { display: flex; gap: .75rem; text-align: left; align-items: center; }
  .attempt img { width: 6rem; aspect-ratio: 16 / 9; object-fit: cover; border-radius: .25rem; }
  .active { border-color: var(--accent); }
  textarea { font: inherit; color: inherit; background: var(--bg); border: 1px solid var(--border); border-radius: .4rem; padding: .6rem; width: 100%; box-sizing: border-box; }
  .confirm { border: 1px solid var(--danger); padding: 1rem; margin-top: 1rem; }
  @media(max-width: 800px) { .layout { grid-template-columns: 1fr; } }
</style>
