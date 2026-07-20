export type ProviderResponseState = 'pending' | 'success' | 'failed' | 'cancelled';

export function classifyProviderResponse(type: string | undefined): ProviderResponseState {
  if (type === 'success') return 'success';
  if (type === 'error') return 'failed';
  if (type === 'cancel' || type === 'dismiss') return 'cancelled';
  return 'pending';
}
