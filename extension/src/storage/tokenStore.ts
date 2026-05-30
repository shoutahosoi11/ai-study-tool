import type { StoredSettings, StoredToken } from '../types'
import { normalizeApiBaseUrl } from '../utils/url'

const tokenKey = 'extensionToken'
const settingsKey = 'settings'
const trustedStorageErrorMessage = 'このブラウザでは安全なtoken保存をサポートしていません。Chrome 102以降を使ってください。'
let trustedStoragePromise: Promise<void> | undefined

type ChromeStorageArea = Pick<typeof chrome.storage.local, 'get' | 'set' | 'remove'> & {
  setAccessLevel?: typeof chrome.storage.local.setAccessLevel
}

function storageArea(): ChromeStorageArea {
  return chrome.storage.local
}

function getFromStorage<T>(key: string): Promise<T | undefined> {
  return new Promise((resolve, reject) => {
    storageArea().get(key, (result) => {
      const lastError = chrome.runtime.lastError
      if (lastError) {
        reject(new Error(lastError.message))
        return
      }
      resolve(result[key] as T | undefined)
    })
  })
}

function setInStorage(value: Record<string, unknown>): Promise<void> {
  return new Promise((resolve, reject) => {
    storageArea().set(value, () => {
      const lastError = chrome.runtime.lastError
      if (lastError) {
        reject(new Error(lastError.message))
        return
      }
      resolve()
    })
  })
}

function removeFromStorage(key: string): Promise<void> {
  return new Promise((resolve, reject) => {
    storageArea().remove(key, () => {
      const lastError = chrome.runtime.lastError
      if (lastError) {
        reject(new Error(lastError.message))
        return
      }
      resolve()
    })
  })
}

export async function restrictStorageToTrustedContexts(): Promise<void> {
  trustedStoragePromise ??= setTrustedStorageAccess()
  return trustedStoragePromise
}

export function resetTrustedStorageProtectionForTests(): void {
  trustedStoragePromise = undefined
}

async function setTrustedStorageAccess(): Promise<void> {
  const area = storageArea()
  const setAccessLevel = area.setAccessLevel
  if (typeof setAccessLevel !== 'function') {
    throw new Error(trustedStorageErrorMessage)
  }
  await new Promise<void>((resolve, reject) => {
    setAccessLevel.call(area, { accessLevel: 'TRUSTED_CONTEXTS' }, () => {
      const lastError = chrome.runtime.lastError
      if (lastError) {
        reject(new Error(trustedStorageErrorMessage))
        return
      }
      resolve()
    })
  })
}

export async function saveToken(token: string, scopes: string[], expiresAt?: string): Promise<void> {
  const trimmed = token.trim()
  if (!trimmed.startsWith('ext_')) {
    throw new Error('invalid extension token')
  }
  await restrictStorageToTrustedContexts()
  const payload: StoredToken = {
    token: trimmed,
    scopes: [...scopes],
    savedAt: new Date().toISOString(),
  }
  if (expiresAt) {
    payload.expiresAt = expiresAt
  }
  await setInStorage({ [tokenKey]: payload })
}

export function getToken(): Promise<StoredToken | undefined> {
  return getFromStorage<StoredToken>(tokenKey)
}

export function clearToken(): Promise<void> {
  return removeFromStorage(tokenKey)
}

export async function getSettings(): Promise<StoredSettings> {
  return (await getFromStorage<StoredSettings>(settingsKey)) ?? {}
}

export async function saveSettings(settings: StoredSettings): Promise<void> {
  const current = await getSettings()
  const next: StoredSettings = { ...current, ...settings }
  if (settings.apiBaseUrl !== undefined) {
    next.apiBaseUrl = normalizeApiBaseUrl(settings.apiBaseUrl)
  }
  await setInStorage({ [settingsKey]: next })
}

export async function markImportedNow(): Promise<void> {
  await saveSettings({ lastImportAt: new Date().toISOString() })
}
