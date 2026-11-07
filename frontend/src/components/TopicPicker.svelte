<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { TopicService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import type { Set } from '../../bindings/github.com/scmbr/pitch/internal/topics/models';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  import { preferences } from '../lib/preferences.svelte';
  let { disabled = false, onTopic, onBusy }: { disabled?: boolean; onTopic: (topic: string) => void; onBusy: (busy: boolean) => void } = $props();
  const t = i18n.t;
  let sets = $state<Set[]>([]);
  let selected = $state<number[]>([]);
  let avoid = $state(preferences.value?.topicWindow ?? 20);
  let text = $state('');
  let busy = $state(true);
  let error = $state<MessageKey | null>(null);
  let disposed = false;
  $effect(() => onBusy(busy));
  onDestroy(() => { disposed = true; });
  onMount(() => {
    void (async () => {
      try {
        sets = (await TopicService.Sets()) ?? [];
        selected = sets.filter(s => s.builtin && s.language === i18n.language).map(s => s.id);
        await draw();
      } catch { error = 'topics.error'; }
      finally { busy = false; }
    })();
  });
  async function draw() {
    busy = true; error = null;
    try {
      if (!Number.isInteger(avoid) || avoid<0 || avoid>1000) throw Error('invalidWindow');
      const topic = await TopicService.Draw(selected, avoid);
      if (!disposed) { text = topic.text; onTopic(text); }
    } catch { text = ''; onTopic(''); error = selected.length ? 'topics.error' : 'topics.emptySelection'; }
    finally { busy = false; }
  }
  function toggle(id: number, checked: boolean) {
    selected = checked ? [...selected,id] : selected.filter(n=>n!==id);
    void draw();
  }
  async function importTopics() {
    busy = true; error = null;
    try {
      const set = await TopicService.Import(t('topics.importTitle'));
      if (set.id) { sets = (await TopicService.Sets()) ?? []; selected=[set.id]; await draw(); }
    } catch { error = 'topics.importError'; }
    finally { busy = false; }
  }
  export function reroll() { if (!disabled && !busy) void draw(); }
</script>

<section class="topics" aria-label={t('topics.title')}>
  <h2>{t('topics.title')}</h2>
  <fieldset disabled={disabled || busy}>
    <legend>{t('topics.sets')}</legend>
    {#each sets as set}
      <label class="set"><input type="checkbox" checked={selected.includes(set.id)} onchange={e=>toggle(set.id,e.currentTarget.checked)} />
        {set.builtin ? t(set.language === 'ru' ? 'topics.ru' : 'topics.en') : set.name} ({set.count})
      </label>
    {/each}
    <label class="avoid">{t('topics.avoid')}<input type="number" min="0" max="1000" bind:value={avoid} /></label>
  </fieldset>
  <p class="prompt" aria-live="polite">{text || t(busy ? 'topics.loading' : 'topics.emptySelection')}</p>
  <div class="actions"><button onclick={reroll} disabled={disabled || busy || !selected.length}>{t('topics.reroll')}</button><button onclick={importTopics} disabled={disabled || busy}>{t('topics.import')}</button></div>
  <p class="hint">{t('topics.windowHint')}</p>
  {#if error}<p role="alert">{t(error)}</p>{/if}
</section>

<style>
  .topics { border:1px solid var(--border); border-radius:.6rem; padding:1rem; margin:1rem 0; }
  h2 { font-size:1.15rem; margin-top:0; } fieldset { border:0; padding:0; display:flex; gap:1rem; flex-wrap:wrap; align-items:center; }
  legend { font-weight:600; margin-bottom:.5rem; } .set { display:flex; align-items:center; gap:.4rem; } .avoid { display:flex; align-items:center; gap:.5rem; } .avoid input { width:5rem; }
  .prompt { font-size:1.4rem; line-height:1.4; } .actions { display:flex; gap:.75rem; flex-wrap:wrap; }
  .hint { color:var(--muted); font-size:.85rem; }
</style>
