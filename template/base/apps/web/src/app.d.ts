import type { SupportedLocale } from '@__APP_SLUG__/i18n';

declare global {
  namespace App {
    interface Locals {
      locale: SupportedLocale;
    }
  }
}

export {};
