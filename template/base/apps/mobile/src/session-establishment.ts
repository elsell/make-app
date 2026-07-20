import { classifySessionFailure, isSessionFailure, type SessionFailure, type SessionFailureDecision } from '@__APP_SLUG__/client-core';

type SessionCredential = { token: string; expiresAt: string };
type ActivationFailure = { failure: SessionFailure; decision: SessionFailureDecision };

export async function activateExchangedSession(
  session: SessionCredential,
  activate: (session: SessionCredential) => Promise<void>,
  persist: (session: SessionCredential | null) => Promise<void>,
): Promise<ActivationFailure | null> {
  try {
    await activate(session);
    return null;
  } catch (cause) {
    const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
    const decision = classifySessionFailure(failure);
    await persist(decision.discardCredential ? null : session);
    return { failure, decision };
  }
}
