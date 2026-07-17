import { createInstance, type TOptions } from 'i18next';
import en from './locales/en.json';
import es from './locales/es.json';

export const defaultLocale = 'en' as const;
export const supportedLocales = ['en', 'es'] as const;
export type SupportedLocale = (typeof supportedLocales)[number];

type CatalogKey = keyof typeof en;
type PluralKey = CatalogKey extends infer Key
  ? Key extends `${infer Stem}_one`
    ? Stem
    : never
  : never;
export type MessageKey = Exclude<CatalogKey, `${string}_one` | `${string}_other`> | PluralKey;

const resources = {
  en: { translation: en },
  es: { translation: es }
} as const;

export interface Translator {
  readonly locale: SupportedLocale;
  t(key: MessageKey, options?: TOptions): string;
  number(value: number, options?: Intl.NumberFormatOptions): string;
  date(value: Date | number, options?: Intl.DateTimeFormatOptions): string;
}

export function selectLocale(candidates: readonly (string | null | undefined)[]): SupportedLocale {
  for (const candidate of candidates) {
    const base = candidate?.trim().toLowerCase().split('-')[0];
    if (supportedLocales.includes(base as SupportedLocale)) return base as SupportedLocale;
  }
  return defaultLocale;
}

export function createTranslator(candidates: readonly (string | null | undefined)[]): Translator {
  const locale = selectLocale(candidates);
  const instance = createInstance();
  void instance.init({
    lng: locale,
    fallbackLng: defaultLocale,
    resources,
    keySeparator: false,
    initAsync: false,
    interpolation: { escapeValue: false }
  });

  return {
    locale,
    t: (key, options) => instance.t(key, options),
    number: (value, options) => new Intl.NumberFormat(locale, options).format(value),
    date: (value, options) => new Intl.DateTimeFormat(locale, options).format(value)
  };
}
