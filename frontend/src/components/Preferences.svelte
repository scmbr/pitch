<script lang="ts">
  import { onMount } from 'svelte';
  import { SettingsService } from '../../bindings/github.com/scmbr/pitch/internal/app';
  import type { Preferences } from '../../bindings/github.com/scmbr/pitch/internal/settings/models';
  import { preferences } from '../lib/preferences.svelte';
  import { shortcut } from '../lib/shortcuts.js';
  import { i18n, type MessageKey } from '../i18n/index.svelte';
  const t=i18n.t;
  let draft=$state<Preferences|null>(null);
  let devices=$state<MediaDeviceInfo[]>([]);
  let message=$state<MessageKey|null>(null);
  let busy=$state(false);
  let limit=$state(0);
  let autoDelete=$state(false);
  const actions:Record<string,MessageKey>={toggle:'training.pause',skip:'training.skip',abort:'training.abort',topic:'topics.reroll',previous:'projects.previous',next:'projects.next',fullscreen:'settings.fullscreen',history:'history.title'};
  onMount(()=>{void SettingsService.Get().then(v=>{draft=JSON.parse(JSON.stringify(v.preferences));limit=v.storageLimitMB;autoDelete=v.autoDelete;}).catch(()=>message='error.loadFailed');});
  async function discover(){try{devices=await navigator.mediaDevices.enumerateDevices();}catch{message='training.cameraError';}}
  async function save(event:SubmitEvent){
    event.preventDefault();if(!draft)return;busy=true;message=null;
    try{const result=await SettingsService.SavePreferences(draft,limit,autoDelete);if(result.ok){preferences.value=JSON.parse(JSON.stringify(draft));message='settings.saved';}else message=result.errorCode==='preferencesInvalid'?'settings.invalidPreferences':'error.saveFailed';}
    catch{message='error.saveFailed';}finally{busy=false;}
  }
</script>
<section>
  <h2>{t('settings.preferences')}</h2>
  {#if draft}
    <form id="preferencesForm" onsubmit={save}>
      <div class="grid">
        <label>{t('training.preparation')}<input type="number" min="0" max="3600" bind:value={draft.preparation} required/></label>
        <label>{t('training.speech')}<input type="number" min="1" max="86400" bind:value={draft.speech} required/></label>
        <label>{t('topics.avoid')}<input type="number" min="0" max="1000" bind:value={draft.topicWindow} required/></label>
        <label>{t('settings.quality')}<select bind:value={draft.quality}><option value="720p">720p</option><option value="480p">480p</option></select></label>
        <label>{t('settings.theme')}<select bind:value={draft.theme}><option value="auto">{t('settings.language.auto')}</option><option value="light">{t('settings.light')}</option><option value="dark">{t('settings.dark')}</option></select></label>
      </div>
      <label>{t('history.limit')}<input type="number" min="0" max="1048576" bind:value={limit} required/></label>
      <label><span><input type="checkbox" bind:checked={autoDelete}/> {t('history.autoDelete')}</span></label>
      <p>{t('history.limitHint')}</p>
      <button type="button" onclick={discover}>{t('settings.devices')}</button>
      <p>{t('settings.deviceHint')}</p>
      {#each ['videoinput','audioinput'] as kind}
        <label>{t(kind==='videoinput'?'settings.camera':'settings.microphone')}
          <select value={kind==='videoinput'?draft.cameraId:draft.microphoneId} onchange={e=>{if(!draft)return;if(kind==='videoinput')draft.cameraId=e.currentTarget.value;else draft.microphoneId=e.currentTarget.value;}}>
            <option value="">{t('settings.language.auto')}</option>
            {#each devices.filter(d=>d.kind===kind) as device,index}<option value={device.deviceId}>{device.label||String(index+1)}</option>{/each}
            {#if (kind==='videoinput'?draft.cameraId:draft.microphoneId)&&!devices.some(d=>d.deviceId===(kind==='videoinput'?draft!.cameraId:draft!.microphoneId))}<option value={kind==='videoinput'?draft.cameraId:draft.microphoneId}>{t('settings.savedDevice')}</option>{/if}
          </select>
        </label>
      {/each}
      <fieldset><legend>{t('settings.shortcuts')}</legend><p>{t('settings.shortcutHint')}</p>
        {#each Object.entries(actions) as [action,label]}
          <label>{t(label)}<input readonly value={draft.shortcuts?.[action] ?? ''} onkeydown={e=>{if(e.code==='Tab')return;e.preventDefault();e.stopPropagation();const key=shortcut(e);if(draft?.shortcuts&&key)draft.shortcuts[action]=key;}}/></label>
        {/each}
      </fieldset>
      <button class="primary" disabled={busy}>{t('settings.save')}</button>
    </form>
  {/if}
  {#if message}<p role={message==='settings.saved'?'status':'alert'}>{t(message)}</p>{/if}
</section>
<style>
  section{max-width:40rem;margin:0 auto;padding:0 1.5rem 2rem;}label{display:grid;gap:.4rem;margin:.75rem 0;} .grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem;}input,select{min-width:0;max-width:100%;box-sizing:border-box;}fieldset{margin:1.5rem 0;border:1px solid var(--border);}p{font-size:.875rem;color:var(--muted);}@media(max-width:600px){.grid{grid-template-columns:1fr;}}
</style>
