export type StoredSession = { token: string; expiresAt: string };

type RestorationDependencies = {
  read(): Promise<string | null>;
  now(): number;
  refreshLeadMs: number;
  refresh(current: StoredSession): Promise<StoredSession>;
  expiryAdvanced(previous: string, next: string): boolean;
  activate(current: StoredSession, renewable: boolean): Promise<void>;
  handleFailure(cause: unknown, current: StoredSession): Promise<void>;
  handleUnreadable(): Promise<void>;
};

function decodeStoredSession(raw: string): StoredSession {
  let parsed: unknown;
  try { parsed = JSON.parse(raw); }
  catch { throw { kind: 'local_storage', reason: 'malformed' } as const; }
  const candidate = parsed as Partial<StoredSession> | null;
  if (!candidate?.token || !candidate.expiresAt || !Number.isFinite(Date.parse(candidate.expiresAt))) {
    throw { kind: 'local_storage', reason: 'missing_fields' } as const;
  }
  return { token: candidate.token, expiresAt: candidate.expiresAt };
}

export async function restoreStoredSession(dependencies: RestorationDependencies): Promise<void> {
  let raw: string | null | undefined;
  let current: StoredSession | null = null;
  try {
    raw = await dependencies.read();
    if (!raw) return;
    current = decodeStoredSession(raw);
    const now = dependencies.now();
    if (Date.parse(current.expiresAt) <= now) throw { kind: 'expired' } as const;
    let renewable = true;
    if (Date.parse(current.expiresAt) - now < dependencies.refreshLeadMs) {
      const previousExpiry = current.expiresAt;
      current = await dependencies.refresh(current);
      renewable = dependencies.expiryAdvanced(previousExpiry, current.expiresAt);
    }
    await dependencies.activate(current, renewable);
  } catch (cause) {
    if (!current && raw) {
      try { current = decodeStoredSession(raw); }
      catch { await dependencies.handleUnreadable(); return; }
    }
    if (!current) { await dependencies.handleUnreadable(); return; }
    await dependencies.handleFailure(cause, current);
  }
}
