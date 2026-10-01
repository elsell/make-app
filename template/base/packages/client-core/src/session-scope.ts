/** Capture before async work; commit results only while the lease is current. */
export function createSessionScope() {
  let generation = 0;
  let account: string | null = null;
  return Object.freeze({
    replace(accountId: string) {
      if (!accountId.trim()) throw new Error('account identity is required');
      generation++;
      account = accountId;
    },
    clear() { generation++; account = null; },
    capture() {
      const expected = generation;
      const owner = account;
      return Object.freeze({
        accountId: owner,
        current: () => owner !== null && generation === expected && account === owner,
      });
    },
  });
}
