# Expo mobile development

Run `make dev` first, then `make mobile` in another terminal. The checked-in app
uses a custom URI scheme (`__APP_NATIVE_ID__://callback`), so use an Expo development
build for reliable OIDC callbacks. Expo Go is not a production authentication
target and may not own that scheme consistently.

Choose device-reachable public URLs:

- iOS simulator can normally use `http://localhost:8080` and
  `http://localhost:5556/dex`.
- Android Emulator uses `http://10.0.2.2:8080` and
  `http://10.0.2.2:5556/dex`.
- A physical device needs the development computer's LAN address and services
  intentionally bound to that interface. Do not expose the bundled development
  credentials on an untrusted network.

Set `EXPO_PUBLIC_API_URL`, `EXPO_PUBLIC_OIDC_ISSUER`, and
`EXPO_PUBLIC_OIDC_CLIENT_ID` before starting Expo. Register
`__APP_NATIVE_ID__://callback` on the provider's public mobile client. Production iOS
and Android identifiers in `apps/mobile/app.json` use the `--bundle-prefix`
selected at generation time (or the local `com.example` default);
replace them before creating signed builds.

When using a LAN issuer, update the provider issuer and API OIDC issuer together;
issuer comparison is exact. Plain HTTP is development-only and requires the
explicit insecure flags.
