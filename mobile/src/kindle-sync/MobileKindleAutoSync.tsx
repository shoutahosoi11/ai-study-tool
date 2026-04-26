import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AppState, Modal, Platform, Pressable, StyleSheet, Text, View } from 'react-native'
import { isAxiosError } from 'axios'
import { WebView, type WebViewMessageEvent } from 'react-native-webview'

import { importKindleHighlights, type ImportKindleHighlightItem } from '../api/highlights'
import { buildKindleSyncInjectedScript } from './injected-script'

type KindleNotebookBook = {
  id: string
  asin: string
  book_title: string
  book_author: string
  notebook_url?: string
}

type KindleNotebookMessage =
  | { type: 'NOTEBOOK_PROGRESS'; stage?: string; count?: number }
  | { type: 'NOTEBOOK_BOOK_LIST'; books?: KindleNotebookBook[] }
  | { type: 'NOTEBOOK_HIGHLIGHT_DATA'; highlights?: ImportKindleHighlightItem[] }
  | { type: 'NOTEBOOK_ERROR'; error?: string }

export type MobileKindleAutoSyncStatus = {
  state: 'idle' | 'syncing' | 'done' | 'error' | 'auth_required'
  message: string
}

type Props = {
  enabled: boolean
  onImported?: () => void | Promise<void>
  onStatusChange?: (status: MobileKindleAutoSyncStatus) => void
}

const LIST_URL = 'https://read.amazon.co.jp/notebook#mode=list'
const RUN_COOLDOWN_MS = 60 * 1000

function buildSyncURL(asin: string) {
  const normalizedASIN = asin.trim()
  if (!normalizedASIN) {
    return ''
  }

  return `https://read.amazon.co.jp/notebook?asin=${encodeURIComponent(normalizedASIN)}#mode=sync`
}

function normalizeBook(raw: KindleNotebookBook): KindleNotebookBook | null {
  const asin = raw.asin?.trim() ?? ''
  const title = raw.book_title?.trim() ?? ''
  const author = raw.book_author?.trim() ?? ''
  const id = raw.id?.trim() || asin || title
  if (!id || !asin) {
    return null
  }
  return {
    id,
    asin,
    book_title: title,
    book_author: author,
    notebook_url: raw.notebook_url?.trim() || undefined,
  }
}

function toStatusMessage(
  book: KindleNotebookBook | null,
  index: number,
  total: number,
  message: string
): string {
  const prefix = total > 0 ? `${Math.max(index, 0)}/${total}冊` : 'Kindle同期'
  const title = book?.book_title?.trim() || book?.asin?.trim() || ''
  return title ? `${prefix} / ${title} / ${message}` : `${prefix} / ${message}`
}

function mapProgressStage(stage?: string, count?: number) {
  switch (stage) {
    case 'notebook_ready':
      return 'Kindle Notebook に接続しました'
    case 'book_list_found':
      return `本一覧を読み取りました（${count ?? 0}冊）`
    case 'highlight_data_progress':
      return `ハイライトを確認中です（現在 ${count ?? 0}件）`
    case 'highlight_data_found':
      return `ハイライトを読み取りました（${count ?? 0}件）`
    default:
      return '同期を進めています'
  }
}

function mapNotebookError(error?: string) {
  switch (error) {
    case 'NOT_LOGGED_IN':
      return 'モバイル内の Amazon ログインが必要です'
    case 'NOTEBOOK_NOT_REACHED':
      return 'Kindle Notebook に移動できませんでした'
    case 'BOOK_LIST_EMPTY':
      return 'Kindle 本一覧を読み取れませんでした'
    default:
      return 'Kindle 自動同期に失敗しました'
  }
}

export function MobileKindleAutoSync({ enabled, onImported, onStatusChange }: Props) {
  const [sourceURL, setSourceURL] = useState<string | null>(null)
  const [webViewKey, setWebViewKey] = useState(0)
  const [loginModalVisible, setLoginModalVisible] = useState(false)

  const runningRef = useRef(false)
  const lastRunAtRef = useRef(0)
  const booksRef = useRef<KindleNotebookBook[]>([])
  const currentIndexRef = useRef(0)
  const processedRef = useRef(0)
  const syncedRef = useRef(0)
  const failedRef = useRef(0)
  const currentModeRef = useRef<'list' | 'sync'>('list')

  const injectedJavaScript = useMemo(() => buildKindleSyncInjectedScript(), [])

  const emitStatus = useCallback(
    (status: MobileKindleAutoSyncStatus) => {
      onStatusChange?.(status)
    },
    [onStatusChange]
  )

  const finish = useCallback(
    (status: MobileKindleAutoSyncStatus) => {
      runningRef.current = false
      setSourceURL(null)
      setLoginModalVisible(false)
      emitStatus(status)
    },
    [emitStatus]
  )

  const presentLoginModal = useCallback(
    (message = 'Amazon にログインすると、このあと自動で Kindle 同期を続けます') => {
      runningRef.current = true
      setLoginModalVisible(true)
      setSourceURL(LIST_URL)
      setWebViewKey((current) => current + 1)
      emitStatus({
        state: 'auth_required',
        message,
      })
    },
    [emitStatus]
  )

  const reloadSyncedBooks = useCallback(async () => {
    if (!onImported) {
      return
    }
    await onImported()
  }, [onImported])

  const openCurrentBook = useCallback(() => {
    const books = booksRef.current
    const currentBook = books[currentIndexRef.current] ?? null
    if (!currentBook) {
      finish({
        state: failedRef.current === books.length ? 'error' : 'done',
        message:
          syncedRef.current > 0
            ? `モバイル同期が完了しました。${syncedRef.current}冊を更新しました`
            : 'モバイル同期が完了しました。新規ハイライトはありませんでした',
      })
      return
    }

    const nextURL = buildSyncURL(currentBook.asin)
    if (!nextURL) {
      failedRef.current += 1
      processedRef.current += 1
      currentIndexRef.current += 1
      void openCurrentBook()
      return
    }

    currentModeRef.current = 'sync'
    setLoginModalVisible(false)
    setSourceURL(nextURL)
    setWebViewKey((current) => current + 1)
    emitStatus({
      state: 'syncing',
      message: toStatusMessage(currentBook, currentIndexRef.current + 1, books.length, 'Kindle 本を開いています'),
    })
  }, [emitStatus, finish])

  const startSync = useCallback(() => {
    if (!enabled || Platform.OS === 'web' || runningRef.current) {
      return
    }

    const now = Date.now()
    if (now - lastRunAtRef.current < RUN_COOLDOWN_MS) {
      return
    }

    runningRef.current = true
    lastRunAtRef.current = now
    booksRef.current = []
    currentIndexRef.current = 0
    processedRef.current = 0
    syncedRef.current = 0
    failedRef.current = 0
    currentModeRef.current = 'list'
    setLoginModalVisible(false)
    setSourceURL(LIST_URL)
    setWebViewKey((current) => current + 1)
    emitStatus({
      state: 'syncing',
      message: 'Kindle Notebook の本一覧を確認しています',
    })
  }, [enabled, emitStatus])

  useEffect(() => {
    if (!enabled || Platform.OS === 'web') {
      finish({
        state: 'idle',
        message: '',
      })
      return
    }

    startSync()

    const subscription = AppState.addEventListener('change', (nextState) => {
      if (nextState === 'active') {
        startSync()
      }
    })

    return () => {
      subscription.remove()
    }
  }, [enabled, finish, startSync])

  const importCurrentHighlights = useCallback(
    async (highlights: ImportKindleHighlightItem[]) => {
      const books = booksRef.current
      const currentBook = books[currentIndexRef.current] ?? null
      if (!currentBook) {
        finish({
          state: 'error',
          message: '同期対象の Kindle 本が見つかりませんでした',
        })
        return
      }

      emitStatus({
        state: 'syncing',
        message: toStatusMessage(currentBook, currentIndexRef.current + 1, books.length, `保存しています（${highlights.length}件）`),
      })

      try {
        if (highlights.length > 0) {
          await importKindleHighlights({ highlights })
          syncedRef.current += 1
          await reloadSyncedBooks()
        }
      } catch (error) {
        if (!(isAxiosError(error) && error.response?.status === 422)) {
          failedRef.current += 1
        }
      } finally {
        processedRef.current += 1
        currentIndexRef.current += 1
        void openCurrentBook()
      }
    },
    [emitStatus, finish, openCurrentBook, reloadSyncedBooks]
  )

  const handleNotebookError = useCallback(
    async (error?: string) => {
      if (error === 'NOT_LOGGED_IN') {
        presentLoginModal()
        return
      }

      if (currentModeRef.current === 'list') {
        finish({
          state: 'error',
          message: mapNotebookError(error),
        })
        return
      }

      failedRef.current += 1
      processedRef.current += 1
      currentIndexRef.current += 1
      await reloadSyncedBooks()
      void openCurrentBook()
    },
    [finish, openCurrentBook, presentLoginModal, reloadSyncedBooks]
  )

  const handleNavigationStateChange = useCallback(
    (url?: string) => {
      const normalizedURL = (url ?? '').toLowerCase()
      if (!normalizedURL) {
        return
      }

      if (
        normalizedURL.includes('/signin') ||
        normalizedURL.includes('/ap/signin') ||
        normalizedURL.includes('/gp/sign-in')
      ) {
        if (!loginModalVisible) {
          presentLoginModal()
        }
      }
    },
    [loginModalVisible, presentLoginModal]
  )

  const handleMessage = useCallback(
    async (event: WebViewMessageEvent) => {
      let message: KindleNotebookMessage | null = null
      try {
        message = JSON.parse(event.nativeEvent.data) as KindleNotebookMessage
      } catch {
        return
      }
      if (!message) {
        return
      }

      if (message.type === 'NOTEBOOK_PROGRESS') {
        const books = booksRef.current
        const currentBook = books[currentIndexRef.current] ?? null
        const currentIndex = currentModeRef.current === 'list' ? 0 : currentIndexRef.current + 1
        emitStatus({
          state: 'syncing',
          message: toStatusMessage(currentBook, currentIndex, books.length, mapProgressStage(message.stage, message.count)),
        })
        return
      }

      if (message.type === 'NOTEBOOK_BOOK_LIST') {
        const nextBooks = (message.books ?? [])
          .map(normalizeBook)
          .filter((book): book is KindleNotebookBook => book !== null)

        booksRef.current = nextBooks
        currentIndexRef.current = 0
        processedRef.current = 0
        syncedRef.current = 0
        failedRef.current = 0

        if (nextBooks.length === 0) {
          finish({
            state: 'error',
            message: 'Kindle 本一覧を読み取れませんでした',
          })
          return
        }

        void openCurrentBook()
        return
      }

      if (message.type === 'NOTEBOOK_HIGHLIGHT_DATA') {
        await importCurrentHighlights(message.highlights ?? [])
        return
      }

      if (message.type === 'NOTEBOOK_ERROR') {
        await handleNotebookError(message.error)
      }
    },
    [emitStatus, finish, handleNotebookError, importCurrentHighlights, openCurrentBook]
  )

  if (!enabled || Platform.OS === 'web' || !sourceURL) {
    return null
  }

  const webView = (
    <WebView
      key={webViewKey}
      source={{ uri: sourceURL }}
      onMessage={handleMessage}
      onNavigationStateChange={(navigationState) => {
        handleNavigationStateChange(navigationState.url)
      }}
      injectedJavaScript={injectedJavaScript}
      javaScriptEnabled
      domStorageEnabled
      sharedCookiesEnabled
      thirdPartyCookiesEnabled
      originWhitelist={['https://*']}
      style={loginModalVisible ? styles.loginWebView : styles.hiddenWebView}
    />
  )

  return (
    <>
      {!loginModalVisible ? (
        <View style={styles.hiddenWebViewShell} pointerEvents="none">
          {webView}
        </View>
      ) : null}

      <Modal visible={loginModalVisible} animationType="slide" presentationStyle="fullScreen">
        <View style={styles.loginModalPage}>
          <View style={styles.loginModalHeader}>
            <View style={styles.loginModalHeaderCopy}>
              <Text style={styles.loginModalTitle}>Amazon にログイン</Text>
              <Text style={styles.loginModalSubtitle}>
                ログインすると、このあと Kindle Notebook から自動で同期を続けます。
              </Text>
            </View>
            <Pressable
              accessibilityRole="button"
              style={({ pressed }) => [styles.loginModalCloseButton, pressed ? styles.loginModalCloseButtonPressed : null]}
              onPress={() => {
                lastRunAtRef.current = 0
                finish({
                  state: 'auth_required',
                  message: 'Amazon ログインを完了すると、次回起動時から自動で Kindle 同期します',
                })
              }}
            >
              <Text style={styles.loginModalCloseButtonText}>閉じる</Text>
            </Pressable>
          </View>
          <View style={styles.loginModalBody}>{webView}</View>
        </View>
      </Modal>
    </>
  )
}

const styles = StyleSheet.create({
  hiddenWebViewShell: {
    height: 1,
    opacity: 0,
    overflow: 'hidden',
    position: 'absolute',
    width: 1,
  },
  hiddenWebView: {
    height: 1,
    width: 1,
  },
  loginWebView: {
    flex: 1,
  },
  loginModalPage: {
    backgroundColor: '#ffffff',
    flex: 1,
  },
  loginModalHeader: {
    alignItems: 'flex-start',
    borderBottomColor: '#eff3f4',
    borderBottomWidth: 1,
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
    paddingBottom: 14,
    paddingHorizontal: 16,
    paddingTop: 56,
  },
  loginModalHeaderCopy: {
    flex: 1,
    gap: 4,
  },
  loginModalTitle: {
    color: '#0f1419',
    fontSize: 20,
    fontWeight: '700',
  },
  loginModalSubtitle: {
    color: '#536471',
    fontSize: 14,
    lineHeight: 20,
  },
  loginModalCloseButton: {
    alignItems: 'center',
    borderColor: '#d7dbdc',
    borderRadius: 999,
    borderWidth: 1,
    justifyContent: 'center',
    minHeight: 38,
    paddingHorizontal: 14,
  },
  loginModalCloseButtonPressed: {
    opacity: 0.75,
  },
  loginModalCloseButtonText: {
    color: '#0f1419',
    fontSize: 14,
    fontWeight: '600',
  },
  loginModalBody: {
    flex: 1,
  },
})
