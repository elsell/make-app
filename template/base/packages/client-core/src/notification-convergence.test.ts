import assert from 'node:assert/strict';
import test from 'node:test';
import {
  openNotificationConvergenceBrowser,
  type NotificationConvergenceBrowserEnvironment,
} from './notification-convergence.js';

const invoke = (listener: (() => void) | null) => listener?.();
const deliver = (listener: ((message: unknown) => void) | null, message: unknown) => listener?.(message);

test('browser convergence retains focus and visibility fallback when its channel is unavailable', () => {
  let focus: (() => void) | null = null;
  let visibility: (() => void) | null = null;
  let visible = false;
  let resumes = 0;
  const environment: NotificationConvergenceBrowserEnvironment = {
    addFocus(listener) { focus = listener; },
    removeFocus(listener) { if (focus === listener) focus = null; },
    addVisibility(listener) { visibility = listener; },
    removeVisibility(listener) { if (visibility === listener) visibility = null; },
    visible: () => visible,
    openChannel() { throw new Error('broadcast_unavailable'); },
  };
  const browser = openNotificationConvergenceBrowser(() => {}, () => { resumes += 1; }, environment);

  invoke(focus);
  invoke(visibility);
  visible = true;
  invoke(visibility);
  assert.equal(resumes, 2);

  browser.close();
  assert.equal(focus, null);
  assert.equal(visibility, null);
});

test('browser convergence safely owns messages, publishing, and teardown failures', () => {
  let receiveChannel: ((message: unknown) => void) | null = null;
  const received: unknown[] = [];
  const environment: NotificationConvergenceBrowserEnvironment = {
    addFocus() {},
    removeFocus() { throw new Error('focus_cleanup_unavailable'); },
    addVisibility() {},
    removeVisibility() { throw new Error('visibility_cleanup_unavailable'); },
    visible: () => true,
    openChannel(receive) {
      receiveChannel = receive;
      return {
        publish() { throw new Error('channel_closed'); },
        close() { throw new Error('channel_cleanup_unavailable'); },
      };
    },
  };
  const browser = openNotificationConvergenceBrowser((message) => received.push(message), () => {}, environment);

  deliver(receiveChannel, { type: 'notification-history-changed', ownerId: 'user-a' });
  assert.deepEqual(received, [{ type: 'notification-history-changed', ownerId: 'user-a' }]);
  assert.doesNotThrow(() => browser.publish({ type: 'notification-history-changed', ownerId: 'user-a' }));
  assert.doesNotThrow(() => browser.close());
  deliver(receiveChannel, { type: 'notification-history-changed', ownerId: 'user-a' });
  assert.equal(received.length, 1);
});
