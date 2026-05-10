const env = {
  apiBaseURL: process.env.EXPO_PUBLIC_API_BASE_URL ?? '',
  firebaseApiKey: process.env.EXPO_PUBLIC_FIREBASE_API_KEY ?? '',
  firebaseAuthDomain: process.env.EXPO_PUBLIC_FIREBASE_AUTH_DOMAIN ?? '',
  firebaseProjectID: process.env.EXPO_PUBLIC_FIREBASE_PROJECT_ID ?? '',
  firebaseMessagingSenderID: process.env.EXPO_PUBLIC_FIREBASE_MESSAGING_SENDER_ID ?? '',
  firebaseAppID: process.env.EXPO_PUBLIC_FIREBASE_APP_ID ?? '',
  admobRewardedAdUnitID:
    process.env.EXPO_PUBLIC_ADMOB_REWARDED_AD_UNIT_ID ?? 'ca-app-pub-3940256099942544/5224354917',
  admobNativeAdUnitIDFeed:
    process.env.EXPO_PUBLIC_ADMOB_NATIVE_AD_UNIT_ID_FEED ?? 'ca-app-pub-3940256099942544/5224354917',
  admobNativeAdUnitIDPostAnswer:
    process.env.EXPO_PUBLIC_ADMOB_NATIVE_AD_UNIT_ID_POST_ANSWER ?? 'ca-app-pub-3940256099942544/5224354917',
}

function normalizeAPIBaseURL(value: string): string {
  const trimmed = value.trim().replace(/\/+$/, '')
  if (trimmed.endsWith('/api')) {
    return `${trimmed}/v1`
  }
  return trimmed
}

export const apiBaseURL = normalizeAPIBaseURL(env.apiBaseURL)

export const firebaseConfig = {
  apiKey: env.firebaseApiKey.trim(),
  authDomain: env.firebaseAuthDomain.trim(),
  projectId: env.firebaseProjectID.trim(),
  messagingSenderId: env.firebaseMessagingSenderID.trim(),
  appId: env.firebaseAppID.trim(),
}

export function isFirebaseConfigured(): boolean {
  return Object.values(firebaseConfig).every((value) => value.length > 0)
}

export const mobileConfigStatus = {
  ready: isFirebaseConfigured() && apiBaseURL.length > 0,
  missing: [
    !apiBaseURL && 'EXPO_PUBLIC_API_BASE_URL',
    !firebaseConfig.apiKey && 'EXPO_PUBLIC_FIREBASE_API_KEY',
    !firebaseConfig.authDomain && 'EXPO_PUBLIC_FIREBASE_AUTH_DOMAIN',
    !firebaseConfig.projectId && 'EXPO_PUBLIC_FIREBASE_PROJECT_ID',
    !firebaseConfig.messagingSenderId && 'EXPO_PUBLIC_FIREBASE_MESSAGING_SENDER_ID',
    !firebaseConfig.appId && 'EXPO_PUBLIC_FIREBASE_APP_ID',
  ].filter(Boolean),
}

export const admobConfig = {
  rewardedAdUnitID: env.admobRewardedAdUnitID.trim(),
  nativeAdUnitIDFeed: env.admobNativeAdUnitIDFeed.trim(),
  nativeAdUnitIDPostAnswer: env.admobNativeAdUnitIDPostAnswer.trim(),
}
