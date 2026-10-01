export type AsyncMutationLease = Readonly<{
  release(): void;
}>;

export type AsyncMutationBarrier = Readonly<{
  enter(): AsyncMutationLease | null;
  blockAndDrain(): Promise<void>;
  unblock(): void;
}>;

export function createAsyncMutationBarrier(): AsyncMutationBarrier {
  let blocked = false;
  let active = 0;
  let drained: Promise<void> | null = null;
  let resolveDrained: (() => void) | null = null;

  return Object.freeze({
    enter(): AsyncMutationLease | null {
      if (blocked) return null;
      active += 1;
      let released = false;
      return Object.freeze({
        release() {
          if (released) return;
          released = true;
          active -= 1;
          if (active === 0 && resolveDrained) {
            resolveDrained();
            drained = null;
            resolveDrained = null;
          }
        },
      });
    },
    async blockAndDrain(): Promise<void> {
      blocked = true;
      if (active === 0) return;
      if (!drained) {
        drained = new Promise<void>((resolve) => {
          resolveDrained = resolve;
        });
      }
      await drained;
    },
    unblock() {
      blocked = false;
    },
  });
}
