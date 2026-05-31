# Mobile Release Readiness

Use this before iOS or Android release builds.

## Build Inputs

- [ ] `EXPO_PUBLIC_API_BASE_URL` points to the production `/api/v1` endpoint.
- [ ] Firebase config values are from the production Firebase project.
- [ ] `EXPO_PUBLIC_APP_VERSION` matches the release version.
- [ ] iOS bundle version/build number is set.
- [ ] Android version code/name is set.
- [ ] No backend secret is present in any `EXPO_PUBLIC_*` value.

## App Check

- [ ] iOS App Check provider is configured.
  - Preferred: App Attest
  - Fallback where needed: DeviceCheck
- [ ] Android App Check provider is configured.
  - Preferred: Play Integrity
- [ ] Firebase App Check enforcement is expected on backend production.
- [ ] Debug App Check token is not shipped in production builds.

## Backend Compatibility

- [ ] Mobile requests send `X-App-Version`.
- [ ] Mobile requests send `X-Platform`.
- [ ] Server `MIN_SUPPORTED_IOS_VERSION` is set.
- [ ] Server `MIN_SUPPORTED_ANDROID_VERSION` is set.
- [ ] Version gate rejection has been tested with an old version.
- [ ] Missing version/platform behavior is understood.

## Auth And Local Security

- [ ] Firebase ID token auth works on a real device.
- [ ] App Check token is accepted on a real device.
- [ ] Recent-auth flows are tested for sensitive operations.
- [ ] Biometric/local app lock, if added, is treated only as local device
  protection and not as server authentication.

## Billing

Mobile in-app purchase / Play Billing is not implemented in this repository
today. If mobile billing is publicly released, use Apple IAP and Google Play
Billing for mobile purchases; do not rely on Stripe alone for mobile digital
goods.

## Observability

- [ ] Crash/error logging destination is selected.
- [ ] PII and secrets are redacted from crash reports.
- [ ] App Check failures and version-gate failures are monitored server-side.

## Store Review

- [ ] Privacy policy is reachable.
- [ ] Data handling disclosure covers highlights, generated questions, account
  data, and billing state.
- [ ] Test account or review instructions are prepared if needed.
- [ ] Rollout plan accounts for non-instant mobile rollback.

