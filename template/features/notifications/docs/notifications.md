# Optional notification delivery

Install with `make-app feature add notifications`. Core delivery contracts live in
`apps/api/internal/platform/notifications`; HTTP/APNs signing adapters live in its
`adapters` package. No endpoint, database table, push permission or entitlement is
registered implicitly.

Inject an HTTP provider and token source at bootstrap. Configure APNs with
`https://api.push.apple.com` (or Apple's sandbox), Firebase with
`https://fcm.googleapis.com`, or Expo with `https://exp.host`. The endpoint is
configuration, not message data. Only trusted operators may select it; credentialed
requests never follow redirects. APNs requires the exact application topic and
an ES256 token from `NewAPNSAuth`; adapt its `Token()` to the contextual callback.
Firebase accepts an injected OAuth access-token source scoped to Firebase
Messaging. Expo optionally uses its access token. Keep credentials outside Git.

`Delivery` bounds attempts and delays using an injected cancellable waiter.
Accepted means the provider accepted submission, not that a device displayed it.
Expo supports receipt polling. Invalid-device results should retire the device
registration; rejected configuration/payload failures should not be retried.
Provider errors contain no device tokens, credentials or response bodies.

The application must authorize recipient/device registration, persist delivery
intent atomically with its domain change, claim an outbox entry, and acknowledge
it only after the provider result is persisted. A retry may duplicate delivery;
use the notification ID for client deduplication. Never infer recipient access
from possession of a push token. On tap, fetch the referenced object through its
normal authorized API. Include adversarial boundary tests when wiring endpoints.
