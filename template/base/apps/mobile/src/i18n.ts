import { createTranslator } from '@__APP_SLUG__/i18n';

export type DeviceLocaleReader = () => readonly { languageTag: string }[];

export function createDeviceTranslator(readLocales: DeviceLocaleReader) {
  return createTranslator(readLocales().map(({ languageTag }) => languageTag));
}
