import type { ImportErrorCode, ImportHighlightItem, ImportHighlightsResponse, ImportResult, KindleHighlight } from '../types'
import { apiV1BaseUrl } from '../utils/url'

export class ExtensionApiError extends Error {
  readonly code: ImportErrorCode

  constructor(code: ImportErrorCode, message: string) {
    super(message)
    Object.setPrototypeOf(this, new.target.prototype)
    this.name = 'ExtensionApiError'
    this.code = code
  }
}

export type ExtensionApiClientOptions = {
  apiBaseUrl: string
  fetchImpl?: typeof fetch
}

export class ExtensionApiClient {
  private readonly apiBaseUrl: string
  private readonly fetchImpl: typeof fetch

  constructor(options: ExtensionApiClientOptions) {
    this.apiBaseUrl = apiV1BaseUrl(options.apiBaseUrl)
    this.fetchImpl = options.fetchImpl ?? fetch
  }

  async startPairing(): Promise<{ pairing_id: string; user_code: string; expires_at: string }> {
    return this.request('/extension/pairing/start', undefined, { method: 'POST' })
  }

  async pairingStatus(pairingId: string): Promise<{ status: 'pending' | 'approved' | 'used' }> {
    return this.request('/extension/pairing/status', undefined, {
      method: 'POST',
      body: JSON.stringify({ pairing_id: pairingId }),
    })
  }

  async claimPairing(pairingId: string): Promise<{ status: string; token?: string; scopes?: string[]; expires_at?: string }> {
    return this.request('/extension/pairing/claim', undefined, {
      method: 'POST',
      body: JSON.stringify({ pairing_id: pairingId }),
    })
  }

  async importHighlights(token: string, highlights: KindleHighlight[]): Promise<ImportResult> {
    const items = highlights.map(toImportHighlightItem)
    const response = await this.request<ImportHighlightsResponse>('/extension/highlights/import', token, {
      method: 'POST',
      body: JSON.stringify({ highlights: items }),
    })
    const result: ImportResult = {
      ok: true,
      savedCount: response.saved_count ?? 0,
      duplicateCount: response.duplicate_count ?? 0,
      skippedCount: (response.copy_protected_count ?? 0) + (response.invalid_item_count ?? 0),
      queuedCount: response.queued_count ?? 0,
    }
    if (response.warning) {
      result.warning = response.warning
    }
    return result
  }

  async revokeSelf(token: string): Promise<void> {
    await this.request('/extension/tokens/self', token, { method: 'DELETE' })
  }

  private async request<T>(path: string, token: string | undefined, init: RequestInit): Promise<T> {
    const headers = new Headers(init.headers)
    headers.set('Content-Type', 'application/json')
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    let response: Response
    try {
      response = await this.fetchImpl(`${this.apiBaseUrl}${path}`, {
        ...init,
        headers,
      })
    } catch {
      throw new ExtensionApiError('network_error', 'ネットワークエラーが発生しました')
    }

    if (!response.ok) {
      throw errorForStatus(response.status)
    }
    if (response.status === 204) {
      return undefined as T
    }
    return (await response.json()) as T
  }
}

export function toImportHighlightItem(highlight: KindleHighlight): ImportHighlightItem {
  return {
    asin: (highlight.asin ?? '').trim(),
    book_title: highlight.bookTitle.trim(),
    book_author: (highlight.bookAuthor ?? '').trim(),
    content: highlight.content.trim(),
    location: [highlight.location, highlight.page].filter(Boolean).join(' / '),
  }
}

export function errorForStatus(status: number): ExtensionApiError {
  if (status === 401) {
    return new ExtensionApiError('unauthorized', '接続が失効しました。再接続してください。')
  }
  if (status === 403) {
    return new ExtensionApiError('forbidden', '拡張機能の権限が不足しています。')
  }
  if (status === 429) {
    return new ExtensionApiError('rate_limited', '取り込み回数が多すぎます。時間を置いて再試行してください。')
  }
  if (status >= 500) {
    return new ExtensionApiError('server_error', 'サーバー側の一時的なエラーです。')
  }
  return new ExtensionApiError('bad_request', 'リクエストを処理できませんでした。')
}
