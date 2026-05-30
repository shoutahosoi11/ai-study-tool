import axios from 'axios'
import { Platform } from 'react-native'

import { getAppCheckToken } from './app-check'
import { getIdToken } from './auth'
import { apiBaseURL, mobileAppVersion } from '../config'

export const apiClient = axios.create({
  baseURL: apiBaseURL,
  timeout: 30_000,
})

apiClient.interceptors.request.use(async (config) => {
  const token = await getIdToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  const appCheckToken = await getAppCheckToken()
  if (appCheckToken) {
    config.headers['X-Firebase-AppCheck'] = appCheckToken
  }

  if (mobileAppVersion) {
    config.headers['X-App-Version'] = mobileAppVersion
  }
  config.headers['X-Platform'] = mobilePlatform()

  return config
})

export function mobilePlatform(): 'ios' | 'android' | 'unknown' {
  if (Platform.OS === 'ios' || Platform.OS === 'android') {
    return Platform.OS
  }
  return 'unknown'
}
