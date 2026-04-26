import type { ExtensionKindleBook } from '../types/kindle'

export type KindleAutoSyncStatus = 'idle' | 'syncing' | 'done' | 'error'

export type KindleAutoSyncSnapshot = {
  status: KindleAutoSyncStatus
  message: string
  synced_at?: string
  visible_signature?: string
  visible_book_ids?: string[]
  total_book_count?: number
  current_book_index?: number
  current_book_title?: string
  current_book_stage?: string
  current_highlight_count?: number
  processed_book_count?: number
  synced_book_count?: number
  failed_book_count?: number
  saved_count?: number
  duplicate_count?: number
  copy_protected_count?: number
}

const STORAGE_KEY = 'ai-study-tool:kindle-auto-sync:v1'
export const KINDLE_AUTO_SYNC_EVENT = 'ai-study-tool:kindle-auto-sync'
export const KINDLE_AUTO_SYNC_COOLDOWN_MS = 10 * 60 * 1000

export function readKindleAutoSyncSnapshot(): KindleAutoSyncSnapshot | null {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') return null
    return parsed as KindleAutoSyncSnapshot
  } catch {
    return null
  }
}

export function writeKindleAutoSyncSnapshot(snapshot: KindleAutoSyncSnapshot) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot))
  } catch {}

  window.dispatchEvent(
    new CustomEvent(KINDLE_AUTO_SYNC_EVENT, {
      detail: snapshot,
    })
  )
}

export function buildVisibleBookSignature(books: ExtensionKindleBook[]): string {
  return books
    .map(function (book) {
      return [book.id, book.asin, book.book_title, book.book_author]
        .map(function (value) {
          return String(value ?? '').trim()
        })
        .join('::')
    })
    .sort()
    .join('|')
}

export function shouldSkipKindleAutoSync(
  snapshot: KindleAutoSyncSnapshot | null,
  visibleSignature: string,
  now = Date.now()
): boolean {
  if (!snapshot || !snapshot.synced_at || !visibleSignature) {
    return false
  }

  if ((snapshot.visible_signature ?? '') !== visibleSignature) {
    return false
  }

  const syncedAt = new Date(snapshot.synced_at).getTime()
  if (Number.isNaN(syncedAt)) {
    return false
  }

  return now-syncedAt < KINDLE_AUTO_SYNC_COOLDOWN_MS
}
