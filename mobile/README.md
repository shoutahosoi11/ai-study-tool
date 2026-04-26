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
