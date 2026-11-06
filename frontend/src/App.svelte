<script lang="ts">
  import { i18n } from './i18n/index.svelte';
  import Settings from './screens/Settings.svelte';

  $effect(() => {
    document.documentElement.lang = i18n.language;
    document.title = i18n.t('app.title');
  });
</script>

<Settings />
