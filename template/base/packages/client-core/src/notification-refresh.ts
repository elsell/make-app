export type NotificationRefreshLatch = Readonly<{
  request(ownerId: string): Promise<void>;
  dispose(): void;
}>;

/** Signals request authoritative reads. Check current() before committing results. */
export function createNotificationRefreshLatch(
  refresh: (ownerId: string, current: () => boolean) => Promise<void>,
): NotificationRefreshLatch {
  let disposed = false;
  let inFlight: Promise<void> | null = null;
  let pendingOwnerId = '';
  let latestOwner = '';
  let generation = 0;

  const drain = async (initialOwnerId: string) => {
    let ownerId = initialOwnerId;
    let failure: unknown;
    let failed = false;
    while (!disposed && ownerId) {
      pendingOwnerId = '';
      const admittedGeneration = generation;
      try {
        await refresh(ownerId, () => !disposed && admittedGeneration === generation);
        failed = false;
      } catch (cause) { failure = cause; failed = true; }
      ownerId = pendingOwnerId;
    }
    inFlight = null;
    if (failed && !disposed) throw failure;
  };

  return Object.freeze({
    request(ownerId: string) {
      if (disposed || !ownerId) return Promise.resolve();
      if (latestOwner !== ownerId) { latestOwner = ownerId; generation++; }
      if (inFlight) { pendingOwnerId = ownerId; return inFlight; }
      // Publish the promise before invoking user code, which may request again.
      let resolve!: () => void;
      let reject!: (cause: unknown) => void;
      const promise = new Promise<void>((done, fail) => { resolve = done; reject = fail; });
      inFlight = promise;
      void drain(ownerId).then(resolve, reject);
      return promise;
    },
    dispose() { disposed = true; generation++; pendingOwnerId = ''; },
  });
}
