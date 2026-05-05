import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getApiErrorMessage } from '../../api/errors'
import { listBookHighlights, listBookHighlightsByMetadata, updateHighlightExplanation } from '../../api/highlights'
import { listKindleBooks } from '../../api/kindle'
import { createQuestionPost } from '../../api/posts'
import { generateQuestions } from '../../api/questions'
import { getMe } from '../../api/users'
import { useKindleAutoSyncSnapshot } from '../../hooks/useKindleAutoSyncSnapshot'
import { useQuestionSyncSnapshot } from '../../hooks/useQuestionSync'
import { theme } from '../../theme'
import type { Highlight } from '../../types/highlight'
import type { KindleBook } from '../../types/kindle'
import type { Question } from '../../types/question'
import { KindleBookCard } from './KindleBookCard'
import { KindleHighlightListModal } from './KindleHighlightListModal'
import { QuestionQuizSessionModal, type QuestionSharePayload } from './QuestionQuizSessionModal'

type Props = {
  onQuestionsGenerated?: (questions: Question[]) => Promise<void> | void
}

function resolveDefaultQuestionCount(value?: number) {
  return typeof value === 'number' ? value : 3
}

function buildMetadataSourceID(bookTitle: string, bookAuthor: string) {
  const normalizedTitle = bookTitle.trim() || 'shared-book'
  const normalizedAuthor = bookAuthor.trim()
  return `metadata:${normalizedTitle}:${normalizedAuthor}`.slice(0, 200)
}

function getQuestionGenerationErrorMessage(error: unknown) {
  const responseMessage = getApiErrorMessage(error)
  if (responseMessage === 'source text is unavailable') {
    return 'この本の保存済みハイライトが見つかりませんでした'
  }
  if (responseMessage === 'questions are still preparing') {
    return '問題はまだ準備中です。少し待ってからもう一度試してください'
  }
  if (responseMessage === 'question generation failed') {
    return '問題生成に失敗しました。時間を置いてもう一度試してください'
  }
  if (responseMessage) {
    return responseMessage
  }

  return '問題の取得に失敗しました'
}

function getSyncHelperText(snapshot: ReturnType<typeof useKindleAutoSyncSnapshot>) {
  if (!snapshot) {
    return '起動時に Chrome 拡張が Kindle Notebook の現在表示分を自動同期します。'
  }

  if (snapshot.status === 'syncing') {
    return snapshot.message || 'Kindle Notebook の現在表示分を同期しています...'
  }
  if (snapshot.status === 'done') {
    return snapshot.message || '自動同期が完了しました。'
  }
  if (snapshot.status === 'error') {
    return snapshot.message || '自動同期に失敗しました'
  }

  return snapshot.message || '起動時に Chrome 拡張が Kindle Notebook の現在表示分を自動同期します。'
}

function getSyncMetaText(snapshot: ReturnType<typeof useKindleAutoSyncSnapshot>) {
  if (!snapshot?.synced_at) {
    return ''
  }

  const syncedAt = new Date(snapshot.synced_at)
  if (Number.isNaN(syncedAt.getTime())) {
    return ''
  }

  const bookCount = snapshot.synced_book_count ?? 0
  const savedCount = snapshot.saved_count ?? 0
  return `${syncedAt.toLocaleString('ja-JP', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })} に同期 / ${bookCount}冊 / 新規 ${savedCount} 件`
}

function getSyncProgressText(snapshot: ReturnType<typeof useKindleAutoSyncSnapshot>) {
  if (!snapshot) {
    return ''
  }

  const totalCount = snapshot.total_book_count ?? 0
  const currentIndex = snapshot.current_book_index ?? 0
  const processedCount = snapshot.processed_book_count ?? 0
  const syncedCount = snapshot.synced_book_count ?? 0
  const failedCount = snapshot.failed_book_count ?? 0
  const currentBookTitle = snapshot.current_book_title?.trim() ?? ''

  if (snapshot.status === 'syncing' && totalCount > 0) {
    const currentHighlightCount = snapshot.current_highlight_count ?? 0
    const stageLabel = snapshot.current_book_stage?.trim() ? ` / ${snapshot.current_book_stage}` : ''
    const label = currentBookTitle ? ` / いま: ${currentBookTitle}` : (currentIndex > 0 ? ` / ${currentIndex}冊目を処理中` : '')
    const highlightLabel = currentHighlightCount > 0 ? ` / 現在 ${currentHighlightCount}件` : ''
    return `${processedCount}/${totalCount}冊確認済み・成功 ${syncedCount}冊・失敗 ${failedCount}冊${label}${highlightLabel}${stageLabel}`
  }

  if ((snapshot.status === 'done' || snapshot.status === 'error') && totalCount > 0) {
    return `対象 ${totalCount}冊 / 成功 ${syncedCount}冊 / 失敗 ${failedCount}冊 / 重複 ${snapshot.duplicate_count ?? 0}件`
  }

  return ''
}

export function KindleBookSection({ onQuestionsGenerated }: Props) {
  const navigate = useNavigate()
  const syncSnapshot = useKindleAutoSyncSnapshot()
  const questionSyncSnapshot = useQuestionSyncSnapshot()

  const [savedBooks, setSavedBooks] = useState<KindleBook[]>([])
  const [savedLoading, setSavedLoading] = useState(true)
  const [sectionError, setSectionError] = useState('')
  const [sectionSuccess, setSectionSuccess] = useState('')
  const [selectedBook, setSelectedBook] = useState<KindleBook | null>(null)
  const [bookHighlights, setBookHighlights] = useState<Highlight[]>([])
  const [highlightsLoading, setHighlightsLoading] = useState(false)
  const [highlightModalError, setHighlightModalError] = useState('')
  const [savingHighlightId, setSavingHighlightId] = useState('')
  const [generatingBookId, setGeneratingBookId] = useState('')
  const [generateStatusText, setGenerateStatusText] = useState<Record<string, string>>({})
  const [defaultQuestionCount, setDefaultQuestionCount] = useState(3)
  const [quizLoading, setQuizLoading] = useState(false)
  const [quizBookTitle, setQuizBookTitle] = useState('')
  const [quizQuestions, setQuizQuestions] = useState<Question[]>([])

  const previousSyncedAtRef = useRef('')
  const progressRefreshKeyRef = useRef('')
  const questionBookStatusMap = new Map(
    (questionSyncSnapshot.books ?? []).map(function (book) {
      return [book.book_key, book] as const
    })
  )

  function fetchSavedBooks(silent?: boolean) {
    if (!silent) {
      setSavedLoading(true)
    }

    listKindleBooks()
      .then(function (res) {
        setSavedBooks(res.books ?? [])
      })
      .catch(function () {
        if (!silent) {
          setSectionError('Kindle本の取得に失敗しました')
        }
      })
      .finally(function () {
        if (!silent) {
          setSavedLoading(false)
        }
      })
  }

  useEffect(function () {
    fetchSavedBooks()
  }, [])

  useEffect(function () {
    getMe()
      .then(function (me) {
        setDefaultQuestionCount(resolveDefaultQuestionCount(me.default_question_count))
      })
      .catch(function () {
        setDefaultQuestionCount(3)
      })
  }, [])

  useEffect(
    function () {
      if (!syncSnapshot?.synced_at || syncSnapshot.synced_at === previousSyncedAtRef.current) {
        return
      }

      previousSyncedAtRef.current = syncSnapshot.synced_at
      fetchSavedBooks(true)
    },
    [syncSnapshot?.synced_at]
  )

  useEffect(
    function () {
      if (syncSnapshot?.status !== 'syncing') {
        progressRefreshKeyRef.current = ''
        return
      }

      const nextKey = [
        syncSnapshot.processed_book_count ?? 0,
        syncSnapshot.synced_book_count ?? 0,
        syncSnapshot.failed_book_count ?? 0,
        syncSnapshot.saved_count ?? 0,
      ].join(':')

      if (!nextKey || nextKey === progressRefreshKeyRef.current) {
        return
      }

      progressRefreshKeyRef.current = nextKey
      fetchSavedBooks(true)
    },
    [
      syncSnapshot?.failed_book_count,
      syncSnapshot?.processed_book_count,
      syncSnapshot?.saved_count,
      syncSnapshot?.status,
      syncSnapshot?.synced_book_count,
    ]
  )

  async function handleViewHighlights(book: KindleBook) {
    setSectionError('')
    setSectionSuccess('')
    setSelectedBook(book)
    setBookHighlights([])
    setHighlightModalError('')
    setHighlightsLoading(true)

    try {
      const highlights = book.asin
        ? (await listBookHighlights(book.asin)).highlights ?? []
        : (await listBookHighlightsByMetadata(book.book_title, book.book_author)).highlights ?? []
      setBookHighlights(highlights)
      if (highlights.length === 0) {
        setHighlightModalError('保存済みハイライトは見つかりませんでした')
      }
    } catch {
      setHighlightModalError('ハイライト一覧の取得に失敗しました')
    } finally {
      setHighlightsLoading(false)
    }
  }

  async function runGenerate(book: KindleBook) {
    const sourceId = book.asin.trim() || buildMetadataSourceID(book.book_title, book.book_author)

    setSectionError('')
    setSectionSuccess('')
    setGeneratingBookId(book.asin)
    setQuizLoading(true)
    setGenerateStatusText(function (prev) {
      return {
        ...prev,
        [book.asin]: '準備済みの問題を読み込んでいます...',
      }
    })
    setQuizBookTitle(book.book_title || 'Kindle 本')
    setQuizQuestions([])

    try {
      const questions = await generateQuestions('kindle_book', sourceId, {
        questionType: 'multiple_choice',
        questionCount: defaultQuestionCount,
        bookTitle: book.book_title,
        bookAuthor: book.book_author,
      })

      if (questions.length === 0) {
        setSectionError('問題はまだ準備中です。少し待ってからもう一度試してください')
        return
      }

      setGenerateStatusText(function (prev) {
        return {
          ...prev,
          [book.asin]: `${questions.length}件の問題を表示しています...`,
        }
      })

      setQuizQuestions(questions)
      if (onQuestionsGenerated) {
        void onQuestionsGenerated(questions)
      }
      setSectionSuccess(`${questions.length}件の問題を用意しました。`)
    } catch (error) {
      setSectionError(getQuestionGenerationErrorMessage(error))
    } finally {
      setQuizLoading(false)
      setGeneratingBookId('')
      setGenerateStatusText(function (prev) {
        const next = { ...prev }
        delete next[book.asin]
        return next
      })
    }
  }

  async function handleShareQuestionSet(payload: QuestionSharePayload) {
    await createQuestionPost({
      body: payload.body,
      book_title: payload.bookTitle,
      question_count: payload.questionCount,
      questions: payload.questions.map(function (question) {
        return {
          question_id: question.questionId,
          sort_order: question.sortOrder,
          note: question.note,
        }
      }),
      type: 'question',
    })
  }

  async function handleSaveExplanation(highlightId: string, explanation: string) {
    setHighlightModalError('')
    setSavingHighlightId(highlightId)
    try {
      const updated = await updateHighlightExplanation(highlightId, explanation)
      setBookHighlights(function (prev) {
        return prev.map(function (highlight) {
          return highlight.id === highlightId ? updated : highlight
        })
      })
    } catch {
      setHighlightModalError('解説の保存に失敗しました')
    } finally {
      setSavingHighlightId('')
    }
  }

  function closeHighlightModal() {
    setSelectedBook(null)
    setBookHighlights([])
    setHighlightsLoading(false)
    setHighlightModalError('')
    setSavingHighlightId('')
  }

  const syncMetaText = getSyncMetaText(syncSnapshot)
  const syncProgressText = getSyncProgressText(syncSnapshot)

  return (
    <section style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.sm, marginBottom: theme.spacing.lg }}>
      <div
        style={{
          border: `1px solid ${theme.colors.border}`,
          borderRadius: theme.radius.md,
          background: theme.colors.backgroundAlt,
          padding: theme.spacing.md,
          display: 'flex',
          flexDirection: 'column',
          gap: theme.spacing.xs,
        }}
      >
        <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>取り込んだ本から学習</p>
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {getSyncHelperText(syncSnapshot)}
        </p>
        {questionSyncSnapshot.message ? (
          <p style={{ margin: 0, color: questionSyncSnapshot.status === 'error' ? theme.colors.danger : theme.colors.secondary, fontSize: theme.fontSize.xs }}>
            {questionSyncSnapshot.message}
          </p>
        ) : null}
        {syncMetaText ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
            {syncMetaText}
          </p>
        ) : null}
        {syncProgressText ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
            {syncProgressText}
          </p>
        ) : null}
      </div>

      {sectionError ? (
        <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{sectionError}</p>
      ) : null}
      {!sectionError && sectionSuccess ? (
        <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>{sectionSuccess}</p>
      ) : null}

      {savedLoading ? (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>読み込み中...</p>
      ) : savedBooks.length === 0 ? (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          まだ Kindle ハイライトがありません。Chrome 拡張を入れた状態でアプリを開くと、自動で同期されます。
        </p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.sm }}>
          {savedBooks.map(function (book) {
            const sourceId = book.asin.trim() || buildMetadataSourceID(book.book_title, book.book_author)
            const questionBookStatus = questionBookStatusMap.get(sourceId)
            const stock = questionBookStatus?.stock
            const target = questionBookStatus?.target
            const preparing = questionBookStatus?.preparing
            const isPreparing = (stock ?? 0) === 0 && (preparing ?? 0) > 0

            return (
              <KindleBookCard
                key={book.asin}
                book={{
                  id: book.asin,
                  asin: book.asin,
                  book_title: book.book_title,
                  book_author: book.book_author,
                  highlight_count: book.highlight_count,
                }}
                stock={stock}
                target={target}
                preparing={preparing}
                isPreparing={isPreparing}
                isGenerating={generatingBookId === book.asin}
                isViewingHighlights={highlightsLoading && selectedBook?.asin === book.asin}
                onViewHighlights={function () {
                  void handleViewHighlights(book)
                }}
                onGenerate={function () {
                  void runGenerate(book)
                }}
              />
            )
          })}
        </div>
      )}

      {selectedBook ? (
        <KindleHighlightListModal
          bookTitle={selectedBook.book_title}
          highlights={bookHighlights}
          loading={highlightsLoading}
          error={highlightModalError}
          savingHighlightId={savingHighlightId}
          onClose={closeHighlightModal}
          onSaveExplanation={handleSaveExplanation}
        />
      ) : null}

      {(quizLoading || quizQuestions.length > 0) ? (
        <QuestionQuizSessionModal
          bookTitle={quizBookTitle}
          questions={quizQuestions}
          loading={quizLoading}
          sessionMode="generate"
          shareEnabled={!quizLoading && quizQuestions.length > 0}
          onShare={handleShareQuestionSet}
          onShareSuccess={function () {
            setSectionSuccess('タイムラインにポストしました。')
            setQuizLoading(false)
            setQuizQuestions([])
            setQuizBookTitle('')
            navigate('/?tab=timeline')
          }}
          onClose={function () {
            setQuizLoading(false)
            setQuizQuestions([])
            setQuizBookTitle('')
          }}
        />
      ) : null}
    </section>
  )
}
