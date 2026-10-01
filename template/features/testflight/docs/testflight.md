# Optional TestFlight delivery

Install with `make-app feature add testflight`. Configure the protected GitHub
`testflight` environment, with required reviewers if appropriate for your team.
Set variables `MOBILE_API_URL`, `MOBILE_OIDC_ISSUER`, `MOBILE_OIDC_CLIENT_ID`,
`APPLE_TEAM_ID`, and `APPLE_PROVISIONING_PROFILE_NAME`.
Set secrets `BUILD_CERTIFICATE_BASE64`, `P12_PASSWORD`,
`BUILD_PROVISION_PROFILE_BASE64`, `APP_STORE_CONNECT_API_KEY_BASE64`,
`APP_STORE_CONNECT_KEY_ID`, and `APP_STORE_CONNECT_ISSUER_ID`.

Register the generated bundle ID with Apple. Supply an App Store distribution
certificate/profile and a P-256 App Store Connect API key. Keep keys outside Git.
Configure application permissions, entitlements and encryption/export declarations
for your actual product before making a release. Push is not implicitly enabled.

Publish a normal release, then manually run **TestFlight** from `main`, selecting
its stable `vX.Y.Z` tag. The gate requires a published release, main ancestry and
successful CI for its exact commit. It checks out that immutable commit for the
build. The tag must already contain this feature's files. Builds use the locked
CocoaPods/Ruby tooling, an ephemeral keychain, application-only manual signing and
archive identity/version checks. Credentials are cleaned up on failure as well.

For recovery, dispatch the same tag with a new, increasing Apple build number.
Never move the release tag. The default joins the run number and attempt with a dot; use an
explicit monotonically increasing number when recovering old runs or changing
workflow numbering. Apple acceptance/upload is distinct from build processing and
native user-journey verification. Build notes wait for the exact app/version/build
and verify the saved notes. They can be rerun independently with the same script
and environment if Apple processing exceeds the bounded wait.

Profile maintenance is deliberate. Supply a JSON metadata file containing
`teamId`, `bundleId`, and SHA-256 hashes of DER distribution certificates in
`certificateSha256`, derived from your validated existing profile. Set
`APPLE_TEAM_ID` and `APPLE_BUNDLE_ID` independently to the expected identity.
Run `node scripts/maintain-store-profile.mjs inspect metadata.json` to inspect.
Only if the product uses push, run `enable-push metadata.json replacement.mobileprovision`.
It reuses the existing certificate and refuses ambiguous/expired identities.
Validate the replacement with `security cms -D -i replacement.mobileprovision |
python3 scripts/verify-store-profile.py` before deliberately replacing the GitHub
profile secret. The command does not rotate secrets or enable associated domains.

Before the first tagged store build, and whenever native dependencies change, run
`scripts/update-store-pod-lock.sh` on macOS with the production configuration
exported. Review and commit `apps/mobile/Podfile.store.lock`. Release builds restore
that lock after clean prebuild and use `pod install --deployment`; they refuse a
missing or stale lock. The generator cannot manufacture an app-specific reviewed
CocoaPods lock without resolving its native project on macOS.
