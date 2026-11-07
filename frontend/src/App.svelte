<script lang="ts">
  import { i18n } from './i18n/index.svelte';
  import Settings from './screens/Settings.svelte';
  import Preferences from './components/Preferences.svelte';
  import Training from './screens/Training.svelte';
  import History from './screens/History.svelte';
  import Projects from './screens/Projects.svelte';
  import type { Version } from '../bindings/github.com/scmbr/pitch/internal/projects/models';
  import { onMount } from 'svelte';
  import { SettingsService } from '../bindings/github.com/scmbr/pitch/internal/app';
  import { Window } from '@wailsio/runtime';
  import { preferences } from './lib/preferences.svelte';
  import { shortcut } from './lib/shortcuts.js';
  let screen = $state('settings');
  let projectVersion = $state<Version | null>(null);
  let projectName = $state('');
  let historyProject = $state(0);
  let impromptu = $state(false);
  onMount(async () => {
    try { const view=await SettingsService.Get(); i18n.setPreference(view.language); preferences.value=view.preferences; } catch { /* Settings shows the actionable error. */ }
  });

  $effect(() => {
    document.documentElement.lang = i18n.language;
    document.title = i18n.t('app.title');
    document.documentElement.dataset.theme=preferences.value?.theme ?? 'auto';
  });
  function keys(event: KeyboardEvent) {
    if(event.repeat || event.defaultPrevented || !preferences.value) return;
    if(event.target instanceof Element && event.target.closest('input,textarea,select,[contenteditable]')) return;
    if(event.target instanceof Element && event.target.closest('button,a') && ['Space','Enter'].includes(event.code)) return;
    const key=shortcut(event);
    const action=Object.entries(preferences.value.shortcuts ?? {}).find(([,value])=>value===key && key)?.[0];
    if(!action) return;
    event.preventDefault();
    if(action==='history') {historyProject=0;screen='history';}
    else if(action==='fullscreen') void Window.ToggleFullscreen();
    else window.dispatchEvent(new CustomEvent('pitch-action',{detail:action}));
  }
</script>
<svelte:window onkeydown={keys}/>

<nav aria-label={i18n.t('nav.label')}>
  <strong>Pitch</strong>
  <button aria-current={screen === 'projects' ? 'page' : undefined} onclick={() => screen = 'projects'}>{i18n.t('projects.title')}</button>
  <button aria-current={screen === 'training' && !impromptu ? 'page' : undefined} onclick={() => { impromptu=false; screen='training'; }}>{i18n.t('training.title')}</button>
  <button aria-current={screen === 'training' && impromptu ? 'page' : undefined} onclick={() => { projectVersion=null; projectName=''; impromptu=true; screen='training'; }}>{i18n.t('topics.impromptu')}</button>
  <button aria-current={screen === 'history' ? 'page' : undefined} onclick={() => { historyProject = 0; screen = 'history'; }}>{i18n.t('history.title')}</button>
  <button aria-current={screen === 'settings' ? 'page' : undefined} onclick={() => screen = 'settings'}>{i18n.t('settings.title')}</button>
</nav>
{#if screen === 'training'}
  {#key `${projectVersion?.id ?? 0}:${impromptu}`}<Training {projectVersion} {projectName} {impromptu} clearProject={() => { projectVersion = null; projectName = ''; }} />{/key}
{:else if screen === 'history'}<History projectID={historyProject} />
{:else if screen === 'projects'}
  <Projects practice={(v,name) => { projectVersion = v; projectName = name; impromptu=false; screen = 'training'; }} history={id => { historyProject = id; screen = 'history'; }} />
{:else}<Settings /><Preferences />{/if}

<style>
  nav { display: flex; align-items: center; gap: 1rem; padding: 1rem 1.5rem; border-bottom: 1px solid var(--border); }
  strong { margin-right: auto; }
  button[aria-current] { border-color: var(--accent); }
</style>
