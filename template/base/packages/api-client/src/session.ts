export const sessionRefreshLeadMs = 5 * 60 * 1000;

export function sessionRefreshDelay(expiresAt: string, now = Date.now()): number {
  const remaining = Date.parse(expiresAt) - now;
  if (!Number.isFinite(remaining) || remaining <= 0) return 0;
  if (remaining > sessionRefreshLeadMs) return remaining - sessionRefreshLeadMs;
  return Math.min(remaining, Math.max(1_000, Math.floor(remaining / 2)));
}

export function sessionExpiryAdvanced(previousExpiresAt: string, nextExpiresAt: string): boolean {
  const previous = Date.parse(previousExpiresAt);
  const next = Date.parse(nextExpiresAt);
  return Number.isFinite(previous) && Number.isFinite(next) && next > previous;
}
