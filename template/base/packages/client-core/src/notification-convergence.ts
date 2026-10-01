export type NotificationConvergenceBrowser = Readonly<{
  publish(signal: unknown): void;
  close(): void;
}>;

type Listener = () => void;

export type NotificationConvergenceBrowserEnvironment = Readonly<{
  addFocus(listener: Listener): void;
  removeFocus(listener: Listener): void;
  addVisibility(listener: Listener): void;
  removeVisibility(listener: Listener): void;
  visible(): boolean;
  openChannel(receive: (message: unknown) => void): Readonly<{
    publish(signal: unknown): void;
    close(): void;
  }>;
}>;

export function openNotificationConvergenceBrowser(
  receive: (message: unknown) => void,
  resume: () => void,
  environment: NotificationConvergenceBrowserEnvironment,
): NotificationConvergenceBrowser {
  let active = true;
  let focusInstalled = false;
  let visibilityInstalled = false;
  let channel: ReturnType<NotificationConvergenceBrowserEnvironment['openChannel']> | null = null;
  const handleFocus = () => { if (active) resume(); };
  const handleVisibility = () => { if (active && environment.visible()) resume(); };

  try { environment.addFocus(handleFocus); focusInstalled = true; } catch {}
  try { environment.addVisibility(handleVisibility); visibilityInstalled = true; } catch {}
  try { channel = environment.openChannel((message) => { if (active) receive(message); }); } catch {}

  return Object.freeze({
    publish(signal) {
      try { channel?.publish(signal); } catch {}
    },
    close() {
      active = false;
      if (focusInstalled) try { environment.removeFocus(handleFocus); } catch {}
      if (visibilityInstalled) try { environment.removeVisibility(handleVisibility); } catch {}
      try { channel?.close(); } catch {}
      channel = null;
    },
  });
}
