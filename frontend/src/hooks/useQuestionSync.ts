import { useCallback, useEffect, useRef, useState } from 'react'
import { syncQuestionStock, type QuestionStockBook, type QuestionStockSyncResponse } from '../api/questions'

type QuestionSyncStatus = 'idle' | 'syncing' | 'done' | 'error'

export type QuestionSyncSnapshot = QuestionStockSyncResponse & {
  status: QuestionSyncStatus
  message: string
  synced_at?: string
}

const STORAGE_KEY = 'ai-study-tool:question-sync:v1'
export const QUESTION_SYNC_EVENT = 'ai-study-tool:question-sync'
const QUESTION_SYNC_POLL_INTERVAL_MS = 30_000

function emptySnapshot(): QuestionSyncSnapshot {
  return {
    status: 'idle',
    message: '',
    books: [],
    queued_count: 0,
    skipped_due_to_daily_limit: false,
  }
}

export function readQuestionSyncSnapshot(): QuestionSyncSnapshot {
  if (typeof window === 'undefined') {
    return emptySnapshot()
  }

  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return emptySnapshot()
    }
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') {
      return emptySnapshot()
    }
    return {
      ...emptySnapshot(),
      ...(parsed as Partial<QuestionSyncSnapshot>),
      books: Array.isArray((parsed as QuestionSyncSnapshot).books) ? (parsed as QuestionSyncSnapshot).books : [],
    }
  } catch {
    return emptySnapshot()
  }
}

export function writeQuestionSyncSnapshot(snapshot: QuestionSyncSnapshot) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot))
  } catch {}

  window.dispatchEvent(
    new CustomEvent(QUESTION_SYNC_EVENT, {
      detail: snapshot,
    })
  )
}

function hasPreparingBooks(books: QuestionStockBook[]) {
  return books.some(function (book) {
    return (book.preparing ?? 0) > 0
  })
}

function buildDoneMessage(response: QuestionStockSyncResponse) {
  if (response.queued_count > 0) {
    return `${response.queued_count}問の準備を開始しました`
  }
  if (response.skipped_due_to_daily_limit) {
    return '今日の問題生成上限に達しています'
  }

  const preparingCount = response.books.reduce(function (total, book) {
    return total + Math.max(book.preparing ?? 0, 0)
  }, 0)
  if (preparingCount > 0) {
    return `${preparingCount}問を準備しています`
  }

  return '問題ストックは最新です'
}

export function useQuestionSyncSnapshot() {
  const [snapshot, setSnapshot] = useState<QuestionSyncSnapshot>(() => readQuestionSyncSnapshot())

  useEffect(function () {
    function handleUpdate(event: Event) {
      if (event instanceof CustomEvent && event.detail && typeof event.detail === 'object') {
        setSnapshot(event.detail as QuestionSyncSnapshot)
        return
      }

      setSnapshot(readQuestionSyncSnapshot())
    }

    window.addEventListener(QUESTION_SYNC_EVENT, handleUpdate)
    return function () {
      window.removeEventListener(QUESTION_SYNC_EVENT, handleUpdate)
    }
  }, [])

  return snapshot
}

export function useQuestionSync(options: { enabled: boolean }) {
  const { enabled } = options
  const [snapshot, setSnapshot] = useState<QuestionSyncSnapshot>(() => readQuestionSyncSnapshot())

  const runningRef = useRef(false)
  const startedRef = useRef(false)

  const runSync = useCallback(
    async function () {
      if (!enabled || runningRef.current) {
        return
      }

      runningRef.current = true
      const syncingSnapshot: QuestionSyncSnapshot = {
        ...snapshot,
        status: 'syncing',
        message: '問題ストックを確認しています...',
      }
      setSnapshot(syncingSnapshot)
      writeQuestionSyncSnapshot(syncingSnapshot)

      try {
        const response = await syncQuestionStock()
        const nextSnapshot: QuestionSyncSnapshot = {
          ...response,
          status: 'done',
          message: buildDoneMessage(response),
          synced_at: new Date().toISOString(),
        }
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
      } catch (error) {
        const nextSnapshot: QuestionSyncSnapshot = {
          ...snapshot,
          status: 'error',
          message: '問題ストックの同期に失敗しました',
        }
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
        throw error
      } finally {
        runningRef.current = false
      }
    },
    [enabled, snapshot]
  )

  useEffect(
    function () {
      if (!enabled) {
        startedRef.current = false
        const nextSnapshot = emptySnapshot()
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
        return
      }

      if (startedRef.current) {
        return
      }

      startedRef.current = true
      void runSync()
    },
    [enabled, runSync]
  )

  useEffect(
    function () {
      if (!enabled || !hasPreparingBooks(snapshot.books)) {
        return
      }

      function handleFocus() {
        void runSync()
      }

      const intervalId = window.setInterval(function () {
        void runSync()
      }, QUESTION_SYNC_POLL_INTERVAL_MS)

      window.addEventListener('focus', handleFocus)
      return function () {
        window.clearInterval(intervalId)
        window.removeEventListener('focus', handleFocus)
      }
    },
    [enabled, runSync, snapshot.books]
  )

  return {
    snapshot,
    runSync,
  }
}
