import { clientRuntimeConfig, type ClientRuntimeConfig } from '@__APP_SLUG__/client-core';

export function parseWebConfig(value: unknown): ClientRuntimeConfig {
  return clientRuntimeConfig(value);
}
