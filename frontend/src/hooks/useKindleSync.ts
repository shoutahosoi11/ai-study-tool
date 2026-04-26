import { useEffect, useRef, useState } from 'react'
import { getIdToken } from '../api/auth'
import type { ExtensionKindleBook, ImportHighlightsResponse } from '../types/kindle'

const REQUEST_EVENT = 'ai-study-tool:kindle-request'
const RESPONSE_EVENT = 'ai-study-tool:kindle-response'
const BOOKS_CACHE_KEY = 'ai-study-tool:kindle-books:v3'
const SYNC_REQUEST_TIMEOUT_MS = 60_000

export type SyncState = 'idle' | 'syncing' | 'done' | 'error'

export interface SyncResult {
  bookId: string
  state: SyncState
  response?: ImportHighlightsResponse
  resolvedAsin?: string
  error?: string
}

export interface SyncProgressInfo {
  stage: string
  count?: number
  message: string
}

type SyncableBook = {
  id: string
  asin: string
  book_title?: string
  book_author?: string
  notebook_url?: string
}

function normalizeString(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function buildNotebookURLFromASIN(asin: string) {
  const normalizedASIN = normalizeString(asin)
  if (!normalizedASIN) return ''
  return `https://read.amazon.co.jp/notebook?asin=${encodeURIComponent(normalizedASIN)}`
}

function normalizeNotebookURL(notebookURL?: string, asin?: string) {
  const normalizedURL = normalizeString(notebookURL)
  if (normalizedURL) return normalizedURL
  return buildNotebookURLFromASIN(asin ?? '')
}

function normalizeBook(raw: unknown): ExtensionKindleBook | null {
  if (!raw || typeof raw !== 'object') return null

  const book = raw as Record<string, unknown>
  const asin = normalizeString(book.asin)
  const notebookURL = normalizeNotebookURL(normalizeString(book.notebook_url), asin)
  const bookTitle = normalizeString(book.book_title)
  const bookAuthor = normalizeString(book.book_author)
  const id =
    normalizeString(book.id) ||
    asin ||
    bookTitle

  if (!id) return null
  if (!asin && !bookTitle) return null

  return {
    id,
    asin,
    book_title: bookTitle,
    book_author: bookAuthor,
    notebook_url: notebookURL || undefined,
  }
}

function getResolvedAsin(result?: ImportHighlightsResponse) {
  if (!result) return ''

  if (typeof result.resolved_asin === 'string' && result.resolved_asin.trim()) {
    return result.resolved_asin.trim()
  }

  for (const highlight of result.highlights ?? []) {
    const asin = typeof highlight?.asin === 'string' ? highlight.asin.trim() : ''
    if (asin) return asin
  }

  return ''
}

function dispatchBridgeRequest(message: Record<string, unknown>) {
  window.postMessage(message, window.location.origin)
  document.dispatchEvent(
    new CustomEvent(REQUEST_EVENT, {
      detail: JSON.stringify(message),
    })
  )
}

function parseBridgeDetail(detail: unknown) {
  if (!detail) return null
  if (typeof detail === 'string') {
    try {
      return JSON.parse(detail)
    } catch {
      return null
    }
  }
  return detail
}

export function useKindleSync() {
  const [extensionInstalled, setExtensionInstalled] = useState(false)
  const [extensionBooks, setExtensionBooks] = useState<ExtensionKindleBook[]>([])
  const [booksLoading, setBooksLoading] = useState(false)
  const [booksError, setBooksError] = useState('')
  const [booksStatus, setBooksStatus] = useState('')
  const [syncing, setSyncing] = useState<Record<string, SyncState>>({})
  const [syncProgress, setSyncProgress] = useState<Record<string, string>>({})
  const [results, setResults] = useState<Record<string, SyncResult>>({})

  const pendingRef = useRef<Map<string, { type: string; asin?: string }>>(new Map())
  const timeoutRef = useRef<Map<string, number>>(new Map())
  const syncRequestResolversRef = useRef<Map<string, (result: SyncResult) => void>>(new Map())
  const syncProgressListenersRef = useRef<Map<string, (progress: SyncProgressInfo) => void>>(new Map())

  function saveBooksCache(books: ExtensionKindleBook[]) {
    try {
      const normalizedBooks = books
        .map(normalizeBook)
        .filter(function (book): book is ExtensionKindleBook { return book !== null })
      window.localStorage.setItem(BOOKS_CACHE_KEY, JSON.stringify(normalizedBooks))
    } catch {}
  }

  function loadBooksCache() {
    try {
      const raw = window.localStorage.getItem(BOOKS_CACHE_KEY)
      if (!raw) return []
      const parsed = JSON.parse(raw)
      if (!Array.isArray(parsed)) return []
      return parsed
        .map(normalizeBook)
        .filter(function (book): book is ExtensionKindleBook { return book !== null })
    } catch {
      return []
    }
  }

  function clearPendingRequest(requestId?: string) {
    if (!requestId) return

    pendingRef.current.delete(requestId)
    syncProgressListenersRef.current.delete(requestId)
    var timeoutId = timeoutRef.current.get(requestId)
    if (timeoutId !== undefined) {
      window.clearTimeout(timeoutId)
      timeoutRef.current.delete(requestId)
    }
  }

  function setRequestTimeout(requestId: string, onTimeout: () => void) {
    const timeoutId = window.setTimeout(function () {
      clearPendingRequest(requestId)
      onTimeout()
    }, SYNC_REQUEST_TIMEOUT_MS)

    timeoutRef.current.set(requestId, timeoutId)
  }

  function resolveSyncRequest(requestId: string | undefined, result: SyncResult) {
    if (!requestId) return

    const resolve = syncRequestResolversRef.current.get(requestId)
    if (!resolve) return

    syncRequestResolversRef.current.delete(requestId)
    resolve(result)
  }

  useEffect(function () {
    function checkExtension() {
      setExtensionInstalled(
        document.documentElement.getAttribute('data-kindle-sync-extension') === 'installed'
      )
    }

    checkExtension()
    setExtensionBooks(loadBooksCache())

    const observer = new MutationObserver(checkExtension)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['data-kindle-sync-extension'],
    })

    function consumeMessage(raw: unknown) {
      if (!raw || typeof raw !== 'object' || !('type' in raw)) return

      const msg = raw as {
        type: string
        requestId?: string
        books?: ExtensionKindleBook[]
        bookId?: string
        result?: ImportHighlightsResponse
        stage?: string
        count?: number
        error?: string | null
      }

      const { type, requestId } = msg

      if (type === 'LIST_BOOKS_REQUEST' || type === 'SYNC_BOOK_REQUEST') {
        return
      }

      if (requestId && !pendingRef.current.has(requestId)) return

      if (type === 'LIST_BOOKS_RESULT' || type === 'SYNC_BOOK_RESULT') {
        clearPendingRequest(requestId)
      }

      if (type === 'LIST_BOOKS_PROGRESS') {
        if (msg.stage === 'bridge_request_sent') {
          setBooksStatus('拡張の画面ブリッジまでは届いています...')
          return
        }
        if (msg.stage === 'background_received') {
          setBooksStatus('拡張本体が一覧取得リクエストを受け取りました...')
          return
        }
        if (msg.stage === 'tab_opened') {
          setBooksStatus('Kindle ノートページを開いています...')
          return
        }
        if (msg.stage === 'notebook_ready') {
          setBooksStatus('Kindle ノートページに到達しました。本一覧を読み取っています...')
          return
        }
        if (msg.stage === 'book_list_found') {
          setBooksStatus(`本一覧を読み取りました（${msg.count ?? 0}件）`)
          return
        }
        if (msg.stage === 'book_list_empty') {
          setBooksStatus('本一覧の読み取り結果が 0 件でした')
          return
        }
      }

      if (type === 'SYNC_BOOK_PROGRESS' && typeof msg.bookId === 'string') {
        const syncMessage =
          msg.stage === 'background_received'
            ? '拡張本体が同期リクエストを受け取りました...'
            : msg.stage === 'tab_opened'
              ? 'Kindle の本ページを開いています...'
              : msg.stage === 'notebook_ready'
                ? 'Kindle ノートページに到達しました...'
                : msg.stage === 'book_ready'
                  ? '対象の本を確認しました。ハイライトを読み取っています...'
                  : msg.stage === 'book_opened'
                    ? '対象の本を開きました。ハイライトを読み取っています...'
                    : msg.stage === 'hash_check_failed'
                      ? '重複チェックに失敗したため、そのまま保存に進めています...'
                    : msg.stage === 'highlight_data_progress'
                      ? `ハイライトを確認中です（現在 ${msg.count ?? 0}件）...`
                      : msg.stage === 'highlight_data_found'
                        ? `ハイライトを読み取りました（${msg.count ?? 0}件）...`
                        : msg.stage === 'highlight_data_received'
                          ? '読み取ったハイライトをアプリへ保存しています...'
                          : '同期を進めています...'

        setSyncProgress(function (prev) {
          return {
            ...prev,
            [msg.bookId as string]: syncMessage,
          }
        })

        const progressListener = requestId ? syncProgressListenersRef.current.get(requestId) : null
        if (progressListener) {
          progressListener({
            stage: msg.stage ?? 'unknown',
            count: typeof msg.count === 'number' ? msg.count : undefined,
            message: syncMessage,
          })
        }
        return
      }

      if (type === 'LIST_BOOKS_RESULT') {
        setBooksLoading(false)
        if (msg.error) {
          setBooksError(
            msg.error === 'NOT_LOGGED_IN'
              ? 'Amazon にログインしてください'
              : msg.error === 'NOTEBOOK_NOT_REACHED'
                ? 'Amazon のノートページに移動できませんでした'
              : msg.error === 'RUNTIME_UNAVAILABLE'
                ? 'Chrome 拡張の runtime に接続できませんでした。拡張を再読み込みしてください'
              : msg.error === 'BACKGROUND_NO_ACK'
                ? 'Chrome 拡張本体が応答しませんでした。chrome://extensions で Service Worker を確認して拡張を再読み込みしてください'
              : msg.error === 'BACKGROUND_UNREACHABLE' || msg.error === 'Could not establish connection. Receiving end does not exist.'
                ? 'Chrome 拡張本体に接続できませんでした。chrome://extensions で拡張を再読み込みしてください'
              : msg.error === 'BOOK_LIST_EMPTY'
                ? 'Kindle の本一覧を読み取れませんでした。ノートページを開いたまま少し待ってから再試行してください'
              : msg.error === 'UNEXPECTED_PAGE'
                ? 'Amazon のノートページに移動できませんでした'
                : msg.error === 'TAB_CREATE_FAILED'
                  ? 'Kindle 取得用タブを開けませんでした'
              : 'Kindle 本一覧の取得に失敗しました'
          )
          return
        }
        const nextBooks = Array.isArray(msg.books)
          ? msg.books
              .map(normalizeBook)
              .filter(function (book): book is ExtensionKindleBook { return book !== null })
          : []
        setExtensionBooks(nextBooks)
        saveBooksCache(nextBooks)
        setBooksError('')
        setBooksStatus(`本一覧を取得しました（${nextBooks.length}件）`)
        return
      }

      if (type === 'SYNC_BOOK_RESULT' && typeof msg.bookId === 'string') {
        const bookId = msg.bookId
        const resolvedAsin = getResolvedAsin(msg.result ?? undefined)
        const syncResult: SyncResult = {
          bookId,
          state: msg.error ? 'error' : 'done',
          response: msg.result ?? undefined,
          resolvedAsin: resolvedAsin || undefined,
          error: msg.error ?? undefined,
        }
        setSyncing(function (prev) {
          const next = { ...prev }
          delete next[bookId]
          return next
        })
        setSyncProgress(function (prev) {
          const next = { ...prev }
          delete next[bookId]
          return next
        })
        setResults(function (prev) {
          return {
            ...prev,
            [bookId]: syncResult,
          }
        })
        resolveSyncRequest(requestId, syncResult)
      }
    }

    function handleWindowMessage(event: MessageEvent) {
      if (event.origin !== window.location.origin) return
      consumeMessage(event.data)
    }

    function handleBridgeEvent(event: Event) {
      if (!(event instanceof CustomEvent)) return
      consumeMessage(parseBridgeDetail(event.detail))
    }

    window.addEventListener('message', handleWindowMessage)
    document.addEventListener(RESPONSE_EVENT, handleBridgeEvent)
    return function () {
      observer.disconnect()
      window.removeEventListener('message', handleWindowMessage)
      document.removeEventListener(RESPONSE_EVENT, handleBridgeEvent)
      timeoutRef.current.forEach(function (timeoutId) {
        window.clearTimeout(timeoutId)
      })
      timeoutRef.current.clear()
      pendingRef.current.clear()
      syncRequestResolversRef.current.clear()
      syncProgressListenersRef.current.clear()
    }
  }, [])

  function listBooksFromExtension() {
    setBooksLoading(true)
    setBooksError('')
    setBooksStatus('Chrome 拡張に一覧取得を依頼しています...')
    const requestId = crypto.randomUUID()
    pendingRef.current.set(requestId, { type: 'list' })
    setRequestTimeout(requestId, function () {
      setBooksLoading(false)
      setBooksError('Kindle 本一覧の取得がタイムアウトしました。Chrome拡張を再読み込みしてから再試行してください')
    })
    dispatchBridgeRequest({ type: 'LIST_BOOKS_REQUEST', requestId })
  }

  async function syncBook(
    book: SyncableBook,
    options?: {
      onProgress?: (progress: SyncProgressInfo) => void
    }
  ) {
    const asin = normalizeString(book.asin)
    const bookTitle = normalizeString(book.book_title)
    const bookAuthor = normalizeString(book.book_author)
    if (!asin) {
      const identifierErrorResult: SyncResult = {
        bookId: book.id,
        state: 'error',
        error: 'BOOK_IDENTIFIER_MISSING',
      }
      setResults(function (prev) {
        return { ...prev, [book.id]: identifierErrorResult }
      })
      return identifierErrorResult
    }

    setSyncProgress(function (prev) {
      return {
        ...prev,
        [book.id]: '同期を開始しています...',
      }
    })
    setSyncing(function (prev) { return { ...prev, [book.id]: 'syncing' } })
    setResults(function (prev) {
      const next = { ...prev }
      delete next[book.id]
      return next
    })

    const token = await getIdToken()
    if (!token) {
      const unauthenticatedResult: SyncResult = {
        bookId: book.id,
        state: 'error',
        error: 'NOT_AUTHENTICATED',
      }
      setSyncing(function (prev) {
        const next = { ...prev }
        delete next[book.id]
        return next
      })
      setSyncProgress(function (prev) {
        const next = { ...prev }
        delete next[book.id]
        return next
      })
      setResults(function (prev) {
        return { ...prev, [book.id]: unauthenticatedResult }
      })
      return unauthenticatedResult
    }

    const requestId = crypto.randomUUID()
    pendingRef.current.set(requestId, { type: 'sync', asin })
    if (options?.onProgress) {
      syncProgressListenersRef.current.set(requestId, options.onProgress)
    }
    const resultPromise = new Promise<SyncResult>(function (resolve) {
      syncRequestResolversRef.current.set(requestId, resolve)
    })
    setRequestTimeout(requestId, function () {
      const timeoutResult: SyncResult = {
        bookId: book.id,
        state: 'error',
        error: 'SYNC_TIMEOUT',
      }
      setSyncing(function (prev) {
        const next = { ...prev }
        delete next[book.id]
        return next
      })
      setSyncProgress(function (prev) {
        const next = { ...prev }
        delete next[book.id]
        return next
      })
      setResults(function (prev) {
        return {
          ...prev,
          [book.id]: timeoutResult,
        }
      })
      resolveSyncRequest(requestId, timeoutResult)
    })
    dispatchBridgeRequest(
      {
        type: 'SYNC_BOOK_REQUEST',
        requestId,
        bookId: book.id,
        asin,
        bookTitle,
        bookAuthor,
        notebookUrl: '',
        token,
        appOrigin: window.location.origin,
      }
    )
    return resultPromise
  }

  return {
    extensionInstalled,
    extensionBooks,
    booksLoading,
    booksError,
    booksStatus,
    listBooksFromExtension,
    syncing,
    syncProgress,
    results,
    syncBook,
  }
}
