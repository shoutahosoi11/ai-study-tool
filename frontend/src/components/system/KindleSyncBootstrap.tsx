import { useEffect, useRef, useState } from 'react'
import { onAuthChanged } from '../../api/auth'
import { useKindleSync } from '../../hooks/useKindleSync'
import { useQuestionSync } from '../../hooks/useQuestionSync'
import {
  buildVisibleBookSignature,
  readKindleAutoSyncSnapshot,
  shouldSkipKindleAutoSync,
  writeKindleAutoSyncSnapshot,
} from '../../lib/kindleAutoSync'
import type { ExtensionKindleBook } from '../../types/kindle'

function filterSyncableBooks(books: ExtensionKindleBook[]) {
  return books.filter(function (book) {
    return Boolean(book.asin.trim())
  })
}

export function KindleSyncBootstrap() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const { runSync: runQuestionSync } = useQuestionSync({ enabled: isAuthenticated })
  const {
    extensionInstalled,
    extensionBooks,
    booksLoading,
    booksError,
    booksStatus,
    listBooksFromExtension,
    syncBook,
  } = useKindleSync()

  const requestedListRef = useRef(false)
  const runningSignatureRef = useRef('')

  useEffect(function () {
    return onAuthChanged(function (user) {
      setIsAuthenticated(Boolean(user))
    })
  }, [])

  useEffect(
    function () {
      if (!isAuthenticated) {
        return
      }

      if (!extensionInstalled) {
        writeKindleAutoSyncSnapshot({
          status: 'idle',
          message: 'Kindle 自動同期には Chrome 拡張が必要です',
        })
        return
      }

      if (requestedListRef.current) {
        return
      }

      requestedListRef.current = true
      writeKindleAutoSyncSnapshot({
        status: 'syncing',
        message: 'Kindle Notebook の現在表示分を確認しています...',
      })
      listBooksFromExtension()
    },
    [extensionInstalled, isAuthenticated]
  )

  useEffect(
    function () {
      if (!isAuthenticated || !extensionInstalled || !requestedListRef.current || booksLoading) {
        return
      }

      if (booksError) {
        writeKindleAutoSyncSnapshot({
          status: 'error',
          message: booksError,
        })
        return
      }

      const syncableBooks = filterSyncableBooks(extensionBooks)
      const visibleSignature = buildVisibleBookSignature(syncableBooks)
      if (!visibleSignature) {
        writeKindleAutoSyncSnapshot({
          status: 'idle',
          message: booksStatus || '同期対象の Kindle 本が見つかりませんでした',
          synced_at: new Date().toISOString(),
          visible_signature: '',
          visible_book_ids: [],
          total_book_count: 0,
          current_book_index: 0,
          processed_book_count: 0,
          synced_book_count: 0,
          failed_book_count: 0,
          saved_count: 0,
          duplicate_count: 0,
          copy_protected_count: 0,
        })
        return
      }

      if (runningSignatureRef.current === visibleSignature) {
        return
      }

      const lastSnapshot = readKindleAutoSyncSnapshot()
      if (shouldSkipKindleAutoSync(lastSnapshot, visibleSignature)) {
        return
      }

      runningSignatureRef.current = visibleSignature
      var cancelled = false

      async function run() {
        const totals = {
          saved: 0,
          duplicate: 0,
          copyProtected: 0,
          syncedBooks: 0,
          failedBooks: 0,
        }

        function writeBookProgressSnapshot(
          book: ExtensionKindleBook | null,
          index: number,
          currentHighlightCount: number,
          currentBookStage: string,
          processedBookCount: number,
          message: string
        ) {
          writeKindleAutoSyncSnapshot({
            status: 'syncing',
            message,
            visible_signature: visibleSignature,
            visible_book_ids: syncableBooks.map(function (item) {
              return item.id
            }),
            total_book_count: syncableBooks.length,
            current_book_index: index,
            processed_book_count: processedBookCount,
            current_book_title: book
              ? book.book_title.trim() || book.asin.trim() || `Kindle 本 ${index || processedBookCount}`
              : '',
            current_book_stage: currentBookStage,
            current_highlight_count: currentHighlightCount,
            synced_book_count: totals.syncedBooks,
            failed_book_count: totals.failedBooks,
            saved_count: totals.saved,
            duplicate_count: totals.duplicate,
            copy_protected_count: totals.copyProtected,
          })
        }

        writeKindleAutoSyncSnapshot({
          status: 'syncing',
          message: `${syncableBooks.length}冊の Kindle 本を同期しています...`,
          visible_signature: visibleSignature,
          visible_book_ids: syncableBooks.map(function (book) {
            return book.id
          }),
          total_book_count: syncableBooks.length,
          current_book_index: 0,
          processed_book_count: 0,
          current_book_title: '',
          synced_book_count: 0,
          failed_book_count: 0,
          saved_count: 0,
          duplicate_count: 0,
          copy_protected_count: 0,
        })

        for (var index = 0; index < syncableBooks.length; index += 1) {
          if (cancelled) {
            return
          }

          const book = syncableBooks[index]
          writeBookProgressSnapshot(book, index + 1, 0, '同期を開始しています...', index, `${index + 1}/${syncableBooks.length}冊を同期しています...`)

          try {
            const result = await syncBook({
              id: book.id,
              asin: book.asin,
              book_title: book.book_title,
              book_author: book.book_author,
            }, {
              onProgress(progress) {
                if (cancelled) {
                  return
                }
                writeBookProgressSnapshot(
                  book,
                  index + 1,
                  progress.count ?? 0,
                  progress.message,
                  index,
                  `${index + 1}/${syncableBooks.length}冊を同期しています...`
                )
              },
            })

            if (result.state === 'done' && result.response) {
              totals.syncedBooks += 1
              totals.saved += result.response.saved_count
              totals.duplicate += result.response.duplicate_count
              totals.copyProtected += result.response.copy_protected_count
            } else {
              totals.failedBooks += 1
            }

            writeBookProgressSnapshot(
              book,
              Math.min(index + 1, syncableBooks.length),
              result.response?.highlights?.length ?? 0,
              result.state === 'done' ? '保存処理まで完了しました' : (result.error ?? '同期に失敗しました'),
              index + 1,
              `${Math.min(index + 1, syncableBooks.length)}/${syncableBooks.length}冊を確認しました...`
            )
          } catch (error) {
            totals.failedBooks += 1
            const errorMessage = error instanceof Error ? error.message : '同期に失敗しました'
            writeBookProgressSnapshot(
              book,
              Math.min(index + 1, syncableBooks.length),
              0,
              errorMessage,
              index + 1,
              `${Math.min(index + 1, syncableBooks.length)}/${syncableBooks.length}冊を確認しました...`
            )
          }
        }

        if (cancelled) {
          return
        }

        try {
          await runQuestionSync()
        } catch {}

        writeKindleAutoSyncSnapshot({
          status: totals.failedBooks === syncableBooks.length ? 'error' : 'done',
          message:
            totals.saved > 0
              ? `自動同期が完了しました。新規 ${totals.saved} 件を取り込みました`
              : '自動同期が完了しました。新規ハイライトはありませんでした',
          synced_at: new Date().toISOString(),
          visible_signature: visibleSignature,
          visible_book_ids: syncableBooks.map(function (book) {
            return book.id
          }),
          total_book_count: syncableBooks.length,
          current_book_index: syncableBooks.length,
          processed_book_count: syncableBooks.length,
          current_book_title: '',
          synced_book_count: totals.syncedBooks,
          failed_book_count: totals.failedBooks,
          saved_count: totals.saved,
          duplicate_count: totals.duplicate,
          copy_protected_count: totals.copyProtected,
        })
      }

      void run()

      return function () {
        cancelled = true
        if (runningSignatureRef.current === visibleSignature) {
          runningSignatureRef.current = ''
        }
      }
    },
    [booksError, booksLoading, booksStatus, extensionBooks, extensionInstalled, isAuthenticated]
  )

  return null
}
