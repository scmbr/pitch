<script lang="ts">
  import { i18n } from './i18n/index.svelte';
  import Settings from './screens/Settings.svelte';
  import Training from './screens/Training.svelte';
  import History from './screens/History.svelte';
  import { onMount } from 'svelte';
  import { SettingsService } from '../bindings/github.com/scmbr/pitch/internal/app';
  let screen = $state('settings');
  onMount(async () => {
    try { i18n.setPreference((await SettingsService.Get()).language); } catch { /* Settings shows the actionable error. */ }
  });

  $effect(() => {
    document.documentElement.lang = i18n.language;
    document.title = i18n.t('app.title');
  });
</script>

<nav aria-label={i18n.t('nav.label')}>
  <strong>Pitch</strong>
  <button aria-current={screen === 'training' ? 'page' : undefined} onclick={() => screen = 'training'}>{i18n.t('training.title')}</button>
  <button aria-current={screen === 'history' ? 'page' : undefined} onclick={() => screen = 'history'}>{i18n.t('history.title')}</button>
  <button aria-current={screen === 'settings' ? 'page' : undefined} onclick={() => screen = 'settings'}>{i18n.t('settings.title')}</button>
</nav>
{#if screen === 'training'}<Training />{:else if screen === 'history'}<History />{:else}<Settings />{/if}

<style>
  nav { display: flex; align-items: center; gap: 1rem; padding: 1rem 1.5rem; border-bottom: 1px solid var(--border); }
  strong { margin-right: auto; }
  button[aria-current] { border-color: var(--accent); }
</style>
