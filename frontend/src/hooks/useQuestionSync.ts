import { useCallback, useEffect, useRef, useState } from 'react'
import { getApiErrorStatus } from '../api/errors'
import { getQuestionStock, syncQuestionStock, type QuestionStockBook, type QuestionStockSyncResponse } from '../api/questions'

type QuestionSyncStatus = 'idle' | 'syncing' | 'done' | 'error'

export type QuestionSyncSnapshot = QuestionStockSyncResponse & {
  status: QuestionSyncStatus
  message: string
  synced_at?: string
}

const STORAGE_KEY = 'ai-study-tool:question-sync:v1'
export const QUESTION_SYNC_EVENT = 'ai-study-tool:question-sync'
const QUESTION_SYNC_POLL_INTERVAL_MS = 30_000
const QUESTION_SYNC_POLL_MAX_INTERVAL_MS = 300_000

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
  const snapshotRef = useRef(snapshot)
  snapshotRef.current = snapshot

  // Polling is paused after a 429 or when the daily generation limit is
  // reached, until the user explicitly triggers a sync again.
  const pollingStoppedRef = useRef(false)
  const pollDelayRef = useRef(QUESTION_SYNC_POLL_INTERVAL_MS)

  const runSync = useCallback(
    async function () {
      if (!enabled || runningRef.current) {
        return
      }

      runningRef.current = true
      pollingStoppedRef.current = false
      pollDelayRef.current = QUESTION_SYNC_POLL_INTERVAL_MS
      const syncingSnapshot: QuestionSyncSnapshot = {
        ...snapshotRef.current,
        status: 'syncing',
        message: '問題ストックを確認しています...',
      }
      setSnapshot(syncingSnapshot)
      writeQuestionSyncSnapshot(syncingSnapshot)

      try {
        const response = await syncQuestionStock()
        if (response.skipped_due_to_daily_limit) {
          pollingStoppedRef.current = true
        }
        const nextSnapshot: QuestionSyncSnapshot = {
          ...response,
          status: 'done',
          message: buildDoneMessage(response),
          synced_at: new Date().toISOString(),
        }
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
      } catch (error) {
        if (getApiErrorStatus(error) === 429) {
          pollingStoppedRef.current = true
        }
        const nextSnapshot: QuestionSyncSnapshot = {
          ...snapshotRef.current,
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
    [enabled]
  )

  // Read-only refresh used by polling (interval + window focus). Never throws.
  const runPoll = useCallback(
    async function () {
      if (!enabled || runningRef.current || pollingStoppedRef.current) {
        return
      }

      runningRef.current = true
      try {
        const response = await getQuestionStock()
        pollDelayRef.current = QUESTION_SYNC_POLL_INTERVAL_MS
        if (response.skipped_due_to_daily_limit) {
          pollingStoppedRef.current = true
        }
        const nextSnapshot: QuestionSyncSnapshot = {
          ...response,
          status: 'done',
          message: buildDoneMessage(response),
          synced_at: new Date().toISOString(),
        }
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
      } catch (error) {
        if (getApiErrorStatus(error) === 429) {
          pollingStoppedRef.current = true
        } else {
          pollDelayRef.current = Math.min(pollDelayRef.current * 2, QUESTION_SYNC_POLL_MAX_INTERVAL_MS)
        }
        const nextSnapshot: QuestionSyncSnapshot = {
          ...snapshotRef.current,
          status: 'error',
          message: '問題ストックの同期に失敗しました',
        }
        setSnapshot(nextSnapshot)
        writeQuestionSyncSnapshot(nextSnapshot)
      } finally {
        runningRef.current = false
      }
    },
    [enabled]
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
      runSync().catch(function () {})
    },
    [enabled, runSync]
  )

  useEffect(
    function () {
      if (!enabled || !hasPreparingBooks(snapshot.books) || pollingStoppedRef.current) {
        return
      }

      let cancelled = false
      let timerId = 0

      function schedule() {
        timerId = window.setTimeout(function () {
          runPoll().then(function () {
            if (!cancelled && !pollingStoppedRef.current) {
              schedule()
            }
          })
        }, pollDelayRef.current)
      }

      function handleFocus() {
        void runPoll()
      }

      schedule()
      window.addEventListener('focus', handleFocus)
      return function () {
        cancelled = true
        window.clearTimeout(timerId)
        window.removeEventListener('focus', handleFocus)
      }
    },
    [enabled, runPoll, snapshot.books]
  )

  return {
    snapshot,
    runSync,
  }
}
