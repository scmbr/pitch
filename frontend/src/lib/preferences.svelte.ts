import type { Preferences } from '../../bindings/github.com/scmbr/pitch/internal/settings/models';
export const preferences = $state<{ value: Preferences | null }>({ value: null });
