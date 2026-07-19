# Native mobile development and delivery

The generated Expo application targets iOS and Android. Its checks intentionally
separate JavaScript export, native project generation, native compilation, a
development client, and a signed store build.

## First development build

Run `make dev`, then create and launch an unsigned development client in another
terminal:

```sh
make mobile-dev-ios      # macOS with Xcode
make mobile-dev-android  # macOS or Linux with Android Studio/SDK
```

The app includes `expo-dev-client`. Later JavaScript-only changes can normally use
`make mobile`, which starts Metro for that installed client. Expo Go is not a
supported authentication target because it does not reliably own the generated
custom callback scheme.

`apps/mobile/eas.json` defines an internal `development` profile. EAS is optional;
local `expo run:ios` and `expo run:android` remain first-class paths. If you use
EAS, use the locked `eas-cli 21.0.2` tooling workspace through the checked package
scripts rather than a floating global CLI or `pnpm dlx`.
The wrapper verifies that EAS runs from `apps/mobile`, so it discovers this
application's `app.json` and `eas.json`.

## Device-reachable development URLs

The root `.env` is loaded by the Make targets. Set all three values together:

```sh
EXPO_PUBLIC_API_URL=http://localhost:8080
EXPO_PUBLIC_OIDC_ISSUER=http://localhost:5556/dex
EXPO_PUBLIC_OIDC_CLIENT_ID=__APP_SLUG__-mobile
```

- An iOS simulator normally reaches host services through `localhost`.
- The Android emulator normally uses `10.0.2.2` instead of `localhost`.
- A physical device needs the development computer's LAN address and services
  intentionally bound to that interface. Never expose bundled development
  credentials on an untrusted network.

Register `__APP_NATIVE_ID__://callback` on the provider's public native client.
Validation proves this scheme, the iOS bundle identifier, and Android package
share the same generated native identity. When changing a LAN issuer, change the
provider issuer and API issuer together. Plain HTTP is development-only.

## What each check proves

```sh
make mobile-validate       # Expo Doctor, dependency compatibility, config identity
make mobile-export         # iOS and Android Metro/Expo bundles only
make mobile-prebuild       # clean generated iOS and Android native projects
make mobile-build-android  # unsigned Android debug APK through Gradle
make mobile-build-ios      # unsigned iOS simulator app through Xcode
```

Bundle export is useful but is not a native build. Generated CI compiles Android
on Linux. It compiles iOS on macOS for trusted pushes to `main` and manual runs;
pull requests perform platform-neutral validation without signing access.

## Release configuration

Before the first store build:

1. Replace the neutral files under `apps/mobile/assets/` with product artwork.
2. Set `expo.version`, `ios.buildNumber`, and `android.versionCode` deliberately.
3. Review every native permission emitted by clean prebuild. Add only permissions
   justified by a product spec and store disclosure.
4. Decide whether to enable Expo Updates. It is disabled by default; runtime
   version follows application version for an explicit compatibility boundary.
5. Configure production HTTPS endpoints, callback URIs, Expo ownership, store
   metadata, signing credentials, and access controls outside Git. Never commit
   signing keys or provider client secrets.

The guarded commands refuse missing, local, or non-HTTPS production endpoints:

```sh
make mobile-release-ios
make mobile-release-android
```

These invoke the checked production EAS profile. Store publication is not
automatic; review and submit signed artifacts deliberately.
The Expo SDK 55 profile selects the reviewed
`ubuntu-24.04-jdk-17-ndk-r27b-sdk-55` Android image and
`macos-sequoia-15.6-xcode-26.2` iOS image. Local iOS builds install the exact
CocoaPods 1.16.2 graph from `Gemfile.lock`; generated CI pins Ruby 3.2.9 and the
JDK patch as well. The dependency-age and OSV gates inspect every locked Ruby
gem, and `make-app doctor` checks the Ruby/Bundler versions used by local builds.

The production profile selects the EAS environment named `production`. Configure
`EXPO_PUBLIC_API_URL`, `EXPO_PUBLIC_OIDC_ISSUER`, and
`EXPO_PUBLIC_OIDC_CLIENT_ID` in that EAS environment before building. They are
public application configuration, not secrets, but the API and issuer must use
HTTPS. The generated application sets `EXPO_PUBLIC_APP_ENV=production` in the
profile and deliberately fails bundling if any production endpoint is absent;
it cannot silently fall back to localhost.

## Session behavior

Native sessions live in platform secure storage. The shared state machine
distinguishes authenticated online, authenticated temporarily offline,
authentication required, and unreadable local storage. Network loss, rate limits,
and server outages retain an otherwise valid credential. Expiry, explicit 401
rejection, revocation, or malformed secure storage removes it. Product-specific
offline data and mutation synchronization remain application concerns.
