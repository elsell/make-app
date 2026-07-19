export type SessionAccessState =
  | 'authenticated_online'
  | 'authenticated_offline'
  | 'authentication_required'
  | 'local_session_unreadable';

export type SessionFailure =
  | { kind: 'network' }
  | { kind: 'http'; status: number }
  | { kind: 'expired' }
  | { kind: 'local_storage'; reason: 'missing_fields' | 'malformed' };

export type SessionFailureDecision = {
  state: SessionAccessState;
  discardCredential: boolean;
  retryable: boolean;
};

export type SessionCredential = { token: string; expiresAt: string };
export type SessionRefreshResponse = { ok: boolean; status: number; json(): Promise<unknown> };

export function classifySessionFailure(failure: SessionFailure): SessionFailureDecision {
  if (failure.kind === 'expired' || (failure.kind === 'http' && failure.status === 401)) {
    return { state: 'authentication_required', discardCredential: true, retryable: false };
  }
  if (failure.kind === 'local_storage') {
    return { state: 'local_session_unreadable', discardCredential: true, retryable: false };
  }
  if (failure.kind === 'network' || failure.status === 408 || failure.status === 429 || failure.status >= 500) {
    return { state: 'authenticated_offline', discardCredential: false, retryable: true };
  }
  return { state: 'authenticated_offline', discardCredential: false, retryable: false };
}

export function sessionFailureFromResponse(status: number): SessionFailure {
  return { kind: 'http', status };
}

export function isSessionFailure(value: unknown): value is SessionFailure {
  if (!value || typeof value !== 'object' || !("kind" in value)) return false;
  return ['network', 'http', 'expired', 'local_storage'].includes(String((value as { kind: unknown }).kind));
}

export async function refreshSessionCredential(
  current: SessionCredential,
  request: (current: SessionCredential) => Promise<SessionRefreshResponse>,
  persist: (replacement: SessionCredential) => Promise<void>,
): Promise<SessionCredential> {
  let response: SessionRefreshResponse;
  try { response = await request(current); }
  catch (cause) {
    if (isSessionFailure(cause)) throw cause;
    throw { kind: 'network' } satisfies SessionFailure;
  }
  if (!response.ok) throw sessionFailureFromResponse(response.status);
  let body: unknown;
  try { body = await response.json(); }
  catch { throw sessionFailureFromResponse(502); }
  const data = (body as { data?: Partial<SessionCredential> } | null)?.data;
  if (!data?.token || !data.expiresAt || !Number.isFinite(Date.parse(data.expiresAt))) {
    throw sessionFailureFromResponse(502);
  }
  const replacement = { token: data.token, expiresAt: data.expiresAt };
  await persist(replacement);
  return replacement;
}

export async function validateSessionCredential<T>(
  current: SessionCredential,
  request: (current: SessionCredential) => Promise<SessionRefreshResponse>,
): Promise<T> {
  let response: SessionRefreshResponse;
  try { response = await request(current); }
  catch (cause) {
    if (isSessionFailure(cause)) throw cause;
    throw { kind: 'network' } satisfies SessionFailure;
  }
  if (!response.ok) throw sessionFailureFromResponse(response.status);
  let body: unknown;
  try { body = await response.json(); }
  catch { throw sessionFailureFromResponse(502); }
  const data = (body as { data?: T } | null)?.data;
  if (!data) throw sessionFailureFromResponse(502);
  return data;
}

export function sessionRetryDelay(attempt: number, expiresAt: string, now = Date.now()): number {
  const untilExpiry = Math.max(0, Date.parse(expiresAt) - now);
  const backoff = Math.min(300_000, 30_000 * (2 ** Math.max(0, Math.min(attempt, 10))));
  return Math.min(backoff, untilExpiry);
}

export function publicEndpointConfig(value: string | undefined, developmentDefault: string, name: string, production: boolean): string {
  const configured = value || (!production ? developmentDefault : '');
  if (!configured) throw new Error(name);
  if (!production) return configured;
  let parsed: URL;
  try { parsed = new URL(configured); } catch { throw new Error(name); }
  const hostname = parsed.hostname.toLowerCase().replace(/^\[|\]$/g, '');
  const privateIPv4Address = (octets: number[]) => octets.length === 4 && octets.every((part) => Number.isInteger(part) && part >= 0 && part <= 255) && (
    octets[0] === 0 || octets[0] === 10 || octets[0] === 127 ||
    (octets[0] === 169 && octets[1] === 254) ||
    (octets[0] === 172 && octets[1] >= 16 && octets[1] <= 31) ||
    (octets[0] === 192 && octets[1] === 168)
  );
  const privateIPv4 = privateIPv4Address(hostname.split('.').map(Number));
  const mapped = /^::ffff:([0-9a-f]{1,4}):([0-9a-f]{1,4})$/.exec(hostname);
  const mappedIPv4 = mapped ? privateIPv4Address([
    Number.parseInt(mapped[1], 16) >>> 8,
    Number.parseInt(mapped[1], 16) & 0xff,
    Number.parseInt(mapped[2], 16) >>> 8,
    Number.parseInt(mapped[2], 16) & 0xff,
  ]) : false;
  const privateIPv6 = hostname.includes(':') && (hostname === '::' || hostname === '::1' || hostname.startsWith('fc') || hostname.startsWith('fd') || /^fe[89ab]/.test(hostname));
  const local = hostname === 'localhost' || hostname.endsWith('.localhost') || hostname.endsWith('.local') || privateIPv4 || mappedIPv4 || privateIPv6;
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password || parsed.search || parsed.hash || local) throw new Error(name);
  return parsed.toString().replace(/\/$/, '');
}

export function publicEnvironmentConfig(value: string | undefined, name: string): 'development' | 'production' {
  const environment = value || 'development';
  if (environment !== 'development' && environment !== 'production') throw new Error(name);
  return environment;
}

export function publicStringConfig(value: string | undefined, developmentDefault: string, name: string, production: boolean): string {
  const configured = value?.trim() || (!production ? developmentDefault : '');
  if (!configured) throw new Error(name);
  return configured;
}

export interface ClientRuntimeConfig {
  readonly environment: 'development' | 'production';
  readonly apiURL: string;
  readonly oidcIssuer: string;
  readonly oidcClientId: string;
}

export function clientRuntimeConfig(value: unknown): ClientRuntimeConfig {
  if (!value || typeof value !== 'object') throw new Error('CLIENT_RUNTIME_CONFIG');
  const candidate = value as Record<string, unknown>;
  if (typeof candidate.environment !== 'string' || !candidate.environment) throw new Error('APP_ENV');
  if (typeof candidate.apiURL !== 'string') throw new Error('API_URL');
  if (typeof candidate.oidcIssuer !== 'string') throw new Error('OIDC_ISSUER');
  if (typeof candidate.oidcClientId !== 'string') throw new Error('OIDC_CLIENT_ID');
  const environment = publicEnvironmentConfig(candidate.environment, 'APP_ENV');
  const production = environment === 'production';
  const config = {
    environment,
    apiURL: publicEndpointConfig(candidate.apiURL, 'http://localhost:8080', 'API_URL', production),
    oidcIssuer: publicEndpointConfig(candidate.oidcIssuer, 'http://localhost:5556/dex', 'OIDC_ISSUER', production),
    oidcClientId: publicStringConfig(candidate.oidcClientId, '__APP_SLUG__-client', 'OIDC_CLIENT_ID', production),
  };
  if (/\s/.test(config.oidcClientId)) throw new Error('OIDC_CLIENT_ID');
  return config;
}
export function isValidSessionCredential(value: unknown): value is SessionCredential {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as Partial<SessionCredential>;
  return typeof candidate.token === 'string' && candidate.token.length > 0 &&
    typeof candidate.expiresAt === 'string' && Number.isFinite(Date.parse(candidate.expiresAt));
}

export function retainedSessionExpiry(value: unknown, fallback: string): string {
  return isValidSessionCredential(value) ? value.expiresAt : fallback;
}
