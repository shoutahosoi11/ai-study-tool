export const DEFAULT_API_BASE_URL = ''
export const API_BASE_URL_PLACEHOLDER = 'https://api.ai-study-tool.com'
// Keep the client-side cap below the backend body/rate limits. The extension sends
// the first page-visible items only because imports are explicit user actions.
export const MAX_IMPORT_HIGHLIGHTS = 100

export type PairingStatus = 'idle' | 'starting' | 'pending' | 'approved' | 'claimed' | 'expired' | 'error'

export type StoredSettings = {
  apiBaseUrl?: string
  lastImportAt?: string
}

export type StoredToken = {
  token: string
  scopes: string[]
  expiresAt?: string
  savedAt: string
}

export type PairingState = {
  pairingId: string
  userCode: string
  expiresAt: string
  status: PairingStatus
}

export type KindleHighlight = {
  asin?: string
  bookTitle: string
  bookAuthor?: string
  content: string
  note?: string
  location?: string
  page?: string
}

export type ExtractResult = {
  highlights: KindleHighlight[]
  totalFound: number
  truncated: boolean
}

export type ImportHighlightItem = {
  asin: string
  book_title: string
  book_author: string
  content: string
  location: string
}

export type ImportHighlightsResponse = {
  queued?: boolean
  queued_count?: number
  saved_count?: number
  duplicate_count?: number
  copy_protected_count?: number
  invalid_item_count?: number
  warning?: string
}

export type ImportResult = {
  ok: true
  savedCount: number
  duplicateCount: number
  skippedCount: number
  queuedCount: number
  warning?: string
}

export type ImportErrorCode = 'missing_token' | 'unauthorized' | 'forbidden' | 'rate_limited' | 'server_error' | 'bad_request' | 'network_error'

export type ImportErrorResult = {
  ok: false
  code: ImportErrorCode
  message: string
}

export type StartImportMessage = {
  type: 'START_IMPORT'
}

export type ImportHighlightsMessage = {
  type: 'IMPORT_HIGHLIGHTS'
  highlights: KindleHighlight[]
}

export type RuntimeMessage = StartImportMessage | ImportHighlightsMessage

export type RuntimeResponse =
  | { ok: true; result?: ImportResult }
  | { ok: false; error: string; code?: ImportErrorCode }
