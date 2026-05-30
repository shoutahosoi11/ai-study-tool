import {
  type AppCheck,
  ReCaptchaV3Provider,
  getToken,
  initializeAppCheck,
} from 'firebase/app-check'

import { firebaseAppCheckConfig } from '../config'
import { getOrCreateFirebaseApp } from './auth'

let appCheckInstance: AppCheck | null = null
let appCheckUnavailable = false

type GlobalWithAppCheckDebug = typeof globalThis & {
  FIREBASE_APPCHECK_DEBUG_TOKEN?: string | boolean
}

function getOrCreateAppCheck(): AppCheck | null {
  if (appCheckUnavailable) {
    return null
  }
  if (appCheckInstance) {
    return appCheckInstance
  }
  if (!firebaseAppCheckConfig.siteKey) {
    return null
  }

  try {
    if (firebaseAppCheckConfig.debugToken) {
      const appCheckGlobal = globalThis as GlobalWithAppCheckDebug
      appCheckGlobal.FIREBASE_APPCHECK_DEBUG_TOKEN = firebaseAppCheckConfig.debugToken
    }
    appCheckInstance = initializeAppCheck(getOrCreateFirebaseApp(), {
      provider: new ReCaptchaV3Provider(firebaseAppCheckConfig.siteKey),
      isTokenAutoRefreshEnabled: true,
    })
    return appCheckInstance
  } catch {
    appCheckUnavailable = true
    return null
  }
}

export async function getAppCheckToken(): Promise<string | null> {
  const appCheck = getOrCreateAppCheck()
  if (!appCheck) {
    return null
  }

  try {
    const result = await getToken(appCheck)
    return result.token || null
  } catch {
    return null
  }
}
