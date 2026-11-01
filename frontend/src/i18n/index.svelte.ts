import { en, type MessageKey } from './en';
import { ru } from './ru';

export type Language = 'ru' | 'en';
export type LanguagePreference = 'auto' | Language;

const dictionaries: Record<Language, Record<MessageKey, string>> = { ru, en };

function systemLanguage(): Language {
  return navigator.language.toLowerCase().startsWith('ru') ? 'ru' : 'en';
}

class I18n {
  preference = $state<LanguagePreference>('auto');
  language = $derived<Language>(this.preference === 'auto' ? systemLanguage() : this.preference);

  t = (key: MessageKey, params: Record<string, string> = {}): string => {
    let text = dictionaries[this.language][key];
    for (const [name, value] of Object.entries(params)) {
      text = text.replaceAll(`{${name}}`, value);
    }
    return text;
  };

  setPreference(value: string): void {
    this.preference = value === 'ru' || value === 'en' ? value : 'auto';
  }
}

export const i18n = new I18n();
export type { MessageKey };
