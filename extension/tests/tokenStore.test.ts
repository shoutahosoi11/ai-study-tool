import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  clearToken,
  getToken,
  resetTrustedStorageProtectionForTests,
  restrictStorageToTrustedContexts,
  saveSettings,
  saveToken,
} from '../src/storage/tokenStore'

const memory = new Map<string, unknown>()

beforeEach(() => {
  memory.clear()
  resetTrustedStorageProtectionForTests()
  vi.stubGlobal('chrome', {
    runtime: { lastError: undefined },
    storage: {
      local: {
        get(key: string, callback: (result: Record<string, unknown>) => void) {
          callback({ [key]: memory.get(key) })
        },
        set(values: Record<string, unknown>, callback: () => void) {
          Object.entries(values).forEach(([key, value]) => memory.set(key, value))
          callback()
        },
        remove(key: string, callback: () => void) {
          memory.delete(key)
          callback()
        },
        setAccessLevel: vi.fn((_value: unknown, callback: () => void) => callback()),
      },
    },
  })
})

describe('tokenStore', () => {
  it('saves, reads, and clears extension token in local storage', async () => {
    await saveToken('ext_test_token', ['highlight:write'], '2026-06-01T00:00:00Z')

    await expect(getToken()).resolves.toMatchObject({
      token: 'ext_test_token',
      scopes: ['highlight:write'],
      expiresAt: '2026-06-01T00:00:00Z',
    })

    await clearToken()
    await expect(getToken()).resolves.toBeUndefined()
  })

  it('rejects non-http API URLs in settings', async () => {
    await expect(saveSettings({ apiBaseUrl: 'javascript:alert(1)' })).rejects.toThrow(/http/)
  })

  it('saves tokens only when storage accessLevel protection succeeds', async () => {
    await saveToken('ext_test_token', ['highlight:write'])
    expect(chrome.storage.local.setAccessLevel).toHaveBeenCalledTimes(1)

    await saveToken('ext_test_token_2', ['highlight:write'])
    expect(chrome.storage.local.setAccessLevel).toHaveBeenCalledTimes(1)
    await expect(getToken()).resolves.toMatchObject({ token: 'ext_test_token_2' })
  })

  it('does not save tokens when storage accessLevel is unsupported', async () => {
    const local = chrome.storage.local as unknown as {
      setAccessLevel?: (value: unknown, callback: () => void) => void
    }
    delete local.setAccessLevel

    await expect(saveToken('ext_test_token', ['highlight:write'])).rejects.toThrow(/Chrome 102/)
    await expect(getToken()).resolves.toBeUndefined()
  })

  it('does not save tokens when storage accessLevel reports lastError', async () => {
    const runtime = chrome.runtime as unknown as { lastError: { message: string } | undefined }
    const local = chrome.storage.local as unknown as {
      setAccessLevel: (value: unknown, callback: () => void) => void
    }
    local.setAccessLevel = vi.fn((_value: unknown, callback: () => void) => {
      runtime.lastError = { message: 'unsupported' }
      callback()
      runtime.lastError = undefined
    })

    await expect(restrictStorageToTrustedContexts()).rejects.toThrow(/Chrome 102/)
    await expect(saveToken('ext_test_token', ['highlight:write'])).rejects.toThrow(/Chrome 102/)
    await expect(getToken()).resolves.toBeUndefined()
  })
})
