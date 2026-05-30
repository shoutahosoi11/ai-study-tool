import { ExtensionApiClient, ExtensionApiError } from './api/client'
import { handleMessage } from './backgroundHandler'
import { getSettings, getToken, clearToken, markImportedNow, restrictStorageToTrustedContexts } from './storage/tokenStore'
import type { RuntimeResponse } from './types'

chrome.runtime.onInstalled.addListener(() => {
  void restrictStorageToTrustedContexts().catch(() => undefined)
})

chrome.runtime.onMessage.addListener((message: unknown, _sender, sendResponse: (response: RuntimeResponse) => void) => {
  void handleMessage(message, {
    importHighlights,
    clearToken,
  }).then(sendResponse)
  return true
})

async function importHighlights(highlights: Parameters<ExtensionApiClient['importHighlights']>[1]) {
  const storedToken = await getToken()
  if (!storedToken?.token) {
    throw new ExtensionApiError('missing_token', '拡張機能をAI Study Toolに接続してください')
  }
  const settings = await getSettings()
  const client = new ExtensionApiClient({ apiBaseUrl: settings.apiBaseUrl ?? '' })
  const result = await client.importHighlights(storedToken.token, highlights)
  await markImportedNow()
  return result
}
