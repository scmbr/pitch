<script lang="ts">
  import { onMount } from 'svelte';
  import { SettingsService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import type { SettingsView } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import { i18n, type MessageKey } from '../i18n/index.svelte';

  const t = i18n.t;

  // Error codes come from Go; every code has a translated message.
  const errorMessages: Record<string, MessageKey> = {
    dataDirEmpty: 'error.dataDirEmpty',
    dataDirNotAbsolute: 'error.dataDirNotAbsolute',
    dataDirNotWritable: 'error.dataDirNotWritable',
    languageUnknown: 'error.languageUnknown',
    saveFailed: 'error.saveFailed',
  };

  type Status = { kind: 'info' | 'error'; key: MessageKey } | null;

  let view = $state<SettingsView | null>(null);
  let dataDir = $state('');
  let language = $state('auto');
  let status = $state<Status>(null);
  let busy = $state(false);

  onMount(async () => {
    try {
      const loaded = await SettingsService.Get();
      view = loaded;
      dataDir = loaded.dataDir;
      language = loaded.language;
      i18n.setPreference(loaded.language);
      if (loaded.warning === 'settingsCorrupt') {
        status = { kind: 'error', key: 'warning.settingsCorrupt' };
      }
    } catch {
      status = { kind: 'error', key: 'error.loadFailed' };
    }
  });

  async function choose() {
    try {
      const picked = await SettingsService.PickDataDir(dataDir, t('settings.dataDir.pickerTitle'));
      if (picked) dataDir = picked;
    } catch {
      status = { kind: 'error', key: 'error.saveFailed' };
    }
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    status = null;
    try {
      const result = await SettingsService.Save(dataDir, language);
      if (result.ok) {
        view = await SettingsService.Get();
        i18n.setPreference(language);
        status = { kind: 'info', key: 'settings.saved' };
      } else {
        status = { kind: 'error', key: errorMessages[result.errorCode] ?? 'error.saveFailed' };
      }
    } catch {
      status = { kind: 'error', key: 'error.saveFailed' };
    } finally {
      busy = false;
    }
  }
</script>

<main>
  <h1>{t('settings.title')}</h1>

  {#if !view && !status}
    <p>{t('settings.loading')}</p>
  {:else if view}
    <form onsubmit={save}>
      <div class="field">
        <label for="dataDir">{t('settings.dataDir.label')}</label>
        <div class="row">
          <input id="dataDir" type="text" bind:value={dataDir} spellcheck="false" autocomplete="off" />
          <button type="button" onclick={choose}>{t('settings.dataDir.choose')}</button>
        </div>
        <p class="hint">{t('settings.dataDir.hint')}</p>
        <p class="hint">{t('settings.dataDir.active', { dir: view.activeDataDir })}</p>
        {#if view.restartRequired}
          <p class="notice">{t('settings.dataDir.restart')}</p>
        {/if}
      </div>

      <div class="field">
        <label for="language">{t('settings.language.label')}</label>
        <select id="language" bind:value={language}>
          <option value="auto">{t('settings.language.auto')}</option>
          <option value="ru">{t('settings.language.ru')}</option>
          <option value="en">{t('settings.language.en')}</option>
        </select>
      </div>

      <button type="submit" class="primary" disabled={busy}>{t('settings.save')}</button>
    </form>
  {/if}

  {#if status}
    <p class={status.kind === 'error' ? 'error' : 'ok'} role={status.kind === 'error' ? 'alert' : 'status'}>
      {t(status.key)}
    </p>
  {/if}
</main>

<style>
  main {
    max-width: 40rem;
    margin: 0 auto;
    padding: 2rem 1.5rem;
  }
  h1 {
    margin: 0 0 1.5rem;
    font-size: 1.5rem;
  }
  .field {
    margin-bottom: 1.5rem;
  }
  label {
    display: block;
    margin-bottom: 0.4rem;
    font-weight: 600;
  }
  .row {
    display: flex;
    gap: 0.5rem;
  }
  input[type='text'] {
    flex: 1;
    min-width: 0;
  }
  .hint {
    margin: 0.4rem 0 0;
    color: var(--muted);
    font-size: 0.875rem;
    overflow-wrap: anywhere;
  }
  .notice {
    margin: 0.4rem 0 0;
    font-size: 0.875rem;
    color: var(--accent);
  }
  .error {
    color: var(--danger);
  }
  .ok {
    color: var(--ok);
  }
</style>
