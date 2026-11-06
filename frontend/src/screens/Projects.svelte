<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { ProjectService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import type { Project, Version } from '../../bindings/github.com/scmbr/pitch/internal/projects/models';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  let { practice, history }: { practice: (v: Version, name: string) => void; history: (id: number) => void } = $props();
  const t = i18n.t;
  let projects = $state<Project[]>([]);
  let selected = $state<Project | null>(null);
  let versions = $state<Version[]>([]);
  let version = $state<Version | null>(null);
  let busy = $state(false);
  let error = $state<MessageKey | null>(null);
  let progress = $state('');
  let page = $state(1);
  let controller: AbortController | null = null;

  async function load() { projects = (await ProjectService.List()) ?? []; }
  onMount(() => { void load().catch(() => error = 'projects.error'); });
  onDestroy(() => controller?.abort());
  async function show(v: Version) {
    version = v; page = 1;
    if (v.hasPDF) {
      controller?.abort(); controller = new AbortController();
      const { prepareSlides } = await import('../lib/slides');
      version = await prepareSlides(v, (p,n) => progress = t('projects.progress', { page: String(p), count: String(n) }), controller.signal);
    }
    versions = (await ProjectService.Versions(v.projectId)) ?? [];
  }
  async function action(task: () => Promise<void>) {
    busy = true; error = null; progress = '';
    try { await task(); }
    catch (e) { if (!controller?.signal.aborted) error = String(e).includes('presentationTooLarge') ? 'projects.tooLarge' : 'projects.error'; }
    finally { busy = false; progress = ''; }
  }
  async function importProject() {
    await action(async () => {
      const v = await ProjectService.Import(t('projects.importTitle'));
      if (!v.id) return;
      await load(); selected = projects.find(p => p.id === v.projectId) ?? null;
      await show(v);
    });
  }
  async function choose(p: Project) {
    await action(async () => {
      selected = p;
      // Refresh hashes the current source. Previously cached versions remain
      // available even if the original file was moved or its disk is offline.
      try { await show(await ProjectService.Refresh(p.id)); }
      catch {
        versions = (await ProjectService.Versions(p.id)) ?? [];
        if (!versions.length) throw Error('presentationMissing');
        await show(versions[0]); error = 'projects.sourceMissing';
      }
    });
  }
  async function attach() {
    if (!version) return;
    await action(async () => {
      await ProjectService.AttachPDF(version!.id, t('projects.pdfTitle'));
      const refreshed = (await ProjectService.Versions(version!.projectId))?.find(v => v.id === version!.id);
      if (refreshed) await show(refreshed);
    });
  }
</script>

<main>
  <h1>{t('projects.title')}</h1>
  <button class="primary" onclick={importProject} disabled={busy}>{t('projects.import')}</button>
  {#if error}<p role="alert">{t(error)}</p>{/if}
  {#if progress}<p role="status">{progress}</p>{/if}
  <div class="layout">
    <section aria-label={t('projects.title')}>
      {#if !projects.length}<p>{t('projects.empty')}</p>{/if}
      {#each projects as p}
        <button class="project" class:active={selected?.id === p.id} onclick={() => choose(p)} disabled={busy}>{p.name}</button>
      {/each}
    </section>
    {#if selected && version}
      <section class="detail" aria-label={selected.name}>
        <h2>{selected.name}</h2>
        <label>{t('projects.version')}
          <select value={version.id} disabled={busy} onchange={e => { const v = versions.find(v => v.id === Number(e.currentTarget.value)); if (v) void action(() => show(v)); }}>
            {#each versions as v}<option value={v.id}>{t('projects.versionNumber', { number: String(v.number) })}</option>{/each}
          </select>
        </label>
        {#if !version.hasPDF}
          <p role="status">{t(version.warning === 'libreOfficeMissing' ? 'projects.noOffice' : 'projects.conversionFailed')}</p>
          <button onclick={attach} disabled={busy}>{t('projects.attachPDF')}</button>
        {:else if version.slideCount > 0}
          <img class="slide" src={`/slides/${version.id}/${page}.png`} alt={t('projects.slide', { page: String(page), count: String(version.slideCount) })} />
          <div class="actions">
            <button onclick={() => page--} disabled={page <= 1}>{t('projects.previous')}</button>
            <span>{page} / {version.slideCount}</span>
            <button onclick={() => page++} disabled={page >= version.slideCount}>{t('projects.next')}</button>
          </div>
          <p class="hint">{t('projects.fidelity')}</p>
          <div class="actions">
            <button class="primary" onclick={() => practice(version!, selected!.name)} disabled={busy}>{t('projects.practice')}</button>
            <button onclick={() => history(selected!.id)} disabled={busy}>{t('history.title')}</button>
          </div>
        {/if}
      </section>
    {/if}
  </div>
</main>

<style>
  main { max-width: 75rem; margin: auto; padding: 2rem 1.5rem; }
  .layout { display: grid; grid-template-columns: minmax(12rem,1fr) 3fr; gap: 2rem; margin-top: 1.5rem; }
  .project { display: block; width: 100%; margin-bottom: .75rem; text-align: left; }
  .active { border-color: var(--accent); } label { display: grid; gap: .4rem; margin-bottom: 1rem; }
  .slide { display: block; width: 100%; max-height: 50vh; object-fit: contain; background: white; }
  .actions { display: flex; gap: 1rem; align-items: center; margin: 1rem 0; flex-wrap: wrap; }
  .hint { color: var(--muted); font-size: .85rem; }
  @media(max-width: 700px) { .layout { grid-template-columns: 1fr; } }
</style>
