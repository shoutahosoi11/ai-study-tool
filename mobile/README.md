# Mobile App

React Native / Expo app for iOS and Android share-sheet intake.

## What works in this scaffold

- Firebase email/password login
- backend user bootstrap via `/api/users/signup`
- shared text draft form
- save to `/api/highlights/share`
- Expo share-intent plugin configured for text and URL sharing

## Setup

1. Copy `.env.example` to `.env`
2. Fill in Firebase config and `EXPO_PUBLIC_API_BASE_URL`
3. Install dependencies

```bash
npm install
```

4. Apply postinstall patches

```bash
npm run postinstall
```

5. Generate native projects and run a custom dev client

```bash
npm run prebuild
npm run ios
npm run android
```

## Important notes

- Share intent does not work in Expo Go.
- `ios/` and `android/` are generated and ignored in git.
- The mobile app expects the backend migration `028_add_highlights_mobile_share_metadata.sql` to be applied.
- API requests call Firebase Auth `getIdToken()` immediately before the request and send it as `Authorization: Bearer <ID token>`. The app does not store Firebase refresh tokens itself.
- API requests send `X-Firebase-AppCheck`, `X-Platform`, and `X-App-Version` when `EXPO_PUBLIC_APP_VERSION` is configured. The app intentionally omits `X-App-Version` instead of defaulting to a fake value; production backend can reject missing versions, and the header is meaningful only together with Firebase ID Token auth and App Check.
- Biometric unlock should be treated only as a local app lock. Do not send biometric success headers as server authorization; sensitive backend operations rely on Firebase `auth_time`.
