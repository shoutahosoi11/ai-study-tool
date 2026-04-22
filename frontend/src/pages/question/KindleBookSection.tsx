import { useEffect, useRef, useState } from 'react'
import axios from 'axios'
import { useNavigate } from 'react-router-dom'
import { listBookHighlights, listBookHighlightsByMetadata, updateHighlightExplanation } from '../../api/highlights'
import { listKindleBooks } from '../../api/kindle'
import { createQuestionPost } from '../../api/posts'
import { generateQuestions } from '../../api/questions'
import { getMe } from '../../api/users'
import { useKindleSync } from '../../hooks/useKindleSync'
import { theme } from '../../theme'
import type { Highlight } from '../../types/highlight'
import type { ExtensionKindleBook, ImportHighlightsResponse, KindleBook } from '../../types/kindle'
import type { Question } from '../../types/question'
import { KindleBookCard } from './KindleBookCard'
import { QuestionQuizSessionModal, type QuestionSharePayload } from './QuestionQuizSessionModal'
import { KindleHighlightListModal } from './KindleHighlightListModal'

type Props = {
  onQuestionsGenerated?: (questions: Question[]) => Promise<void> | void
}

type DisplayBook = {
  id: string
  asin: string
  book_title: string
  book_author: string
  highlight_count: number
  notebook_url?: string
}

function resolveDefaultQuestionCount(value?: number) {
  return typeof value === 'number' ? value : 3
}

function normalizeMatchText(value?: string) {
  return (value ?? '').trim().toLowerCase()
}

function buildStrictBookKey(bookTitle?: string, bookAuthor?: string) {
  const normalizedTitle = normalizeMatchText(bookTitle)
  if (!normalizedTitle) return ''
  return `${normalizedTitle}::${normalizeMatchText(bookAuthor)}`
}

function findSavedBookMatch(book: Pick<DisplayBook, 'asin' | 'book_title' | 'book_author'>, savedBooks: KindleBook[]) {
  if (book.asin) {
    const byASIN = savedBooks.find(function (savedBook) {
      return savedBook.asin === book.asin
    })
    if (byASIN) return byASIN
  }

  if (!book.book_title.trim() || !book.book_author.trim()) {
    return undefined
  }

  const strictKey = buildStrictBookKey(book.book_title, book.book_author)
  if (strictKey) {
    const strictMatches = savedBooks.filter(function (savedBook) {
      return buildStrictBookKey(savedBook.book_title, savedBook.book_author) === strictKey
    })
    if (strictMatches.length === 1) return strictMatches[0]
  }

  return undefined
}

function mergeBooks(
  extensionBooks: ExtensionKindleBook[],
  savedBooks: KindleBook[]
): DisplayBook[] {
  return extensionBooks.map(function (eb) {
    const saved = findSavedBookMatch(
      {
        asin: eb.asin,
        book_title: eb.book_title,
        book_author: eb.book_author,
      },
      savedBooks
    )
    return {
      id: eb.id,
      asin: eb.asin || (saved ? saved.asin : ''),
      book_title: eb.book_title || (saved ? saved.book_title : ''),
      book_author: eb.book_author || (saved ? saved.book_author : ''),
      highlight_count: saved ? saved.highlight_count : 0,
      notebook_url: eb.notebook_url,
    }
  })
}

function getAsinFromNotebookURL(notebookURL?: string) {
  if (!notebookURL) return ''

  try {
    return new URL(notebookURL, window.location.origin).searchParams.get('asin') ?? ''
  } catch {
    return ''
  }
}

function getBookSourceId(
  book: DisplayBook,
  savedBooks: KindleBook[],
  resolvedAsin?: string
) {
  const savedMatch = findSavedBookMatch(book, savedBooks)
  return (
    resolvedAsin ||
    book.asin ||
    (savedMatch ? savedMatch.asin : '') ||
    getAsinFromNotebookURL(book.notebook_url)
  )
}

async function loadHighlightsForBook(
  sourceId: string,
  lookupTitle: string,
  lookupAuthor: string
) {
  if (sourceId) {
    const byAsin = await listBookHighlights(sourceId)
    if ((byAsin.highlights ?? []).length > 0 || !lookupTitle) {
      return byAsin.highlights ?? []
    }
  }

  if (lookupTitle) {
    const byMetadata = await listBookHighlightsByMetadata(lookupTitle, lookupAuthor)
    return byMetadata.highlights ?? []
  }

  return []
}

function getImportedBookMetadata(syncResponse?: ImportHighlightsResponse) {
  const importedHighlight = syncResponse?.highlights?.find(function (highlight) {
    return Boolean((highlight.book_title ?? '').trim()) || Boolean((highlight.book_author ?? '').trim())
  })

  return {
    bookTitle: importedHighlight?.book_title?.trim() ?? '',
    bookAuthor: importedHighlight?.book_author?.trim() ?? '',
  }
}

function getLookupBookMetadata(
  book: DisplayBook,
  savedBooks: KindleBook[],
  syncResponse?: ImportHighlightsResponse
) {
  const savedMatch = findSavedBookMatch(book, savedBooks)
  const importedMetadata = getImportedBookMetadata(syncResponse)

  return {
    bookTitle:
      book.book_title.trim() ||
      savedMatch?.book_title?.trim() ||
      importedMetadata.bookTitle,
    bookAuthor:
      book.book_author.trim() ||
      savedMatch?.book_author?.trim() ||
      importedMetadata.bookAuthor,
  }
}

function getQuestionGenerationErrorMessage(error: unknown) {
  if (axios.isAxiosError(error)) {
    const responseMessage =
      typeof error.response?.data?.message === 'string'
        ? error.response.data.message.trim()
        : ''

    if (responseMessage === 'source text is unavailable') {
      return 'この本の保存済みハイライトが見つかりませんでした'
    }
    if (responseMessage) {
      return responseMessage
    }
  }

  return '問題の作成に失敗しました'
}

export function KindleBookSection({ onQuestionsGenerated }: Props) {
  const navigate = useNavigate()
  const [savedBooks, setSavedBooks] = useState<KindleBook[]>([])
  const [savedLoading, setSavedLoading] = useState(true)
  const [sectionError, setSectionError] = useState('')
  const [sectionSuccess, setSectionSuccess] = useState('')
  const [selectedBook, setSelectedBook] = useState<DisplayBook | null>(null)
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
  const {
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
  } = useKindleSync()

  const previousDoneSignatureRef = useRef('[]')

  function fetchSavedBooks(silent?: boolean) {
    if (!silent) setSavedLoading(true)
    listKindleBooks()
      .then(function (res) {
        setSavedBooks(res.books ?? [])
      })
      .catch(function () {
        if (!silent) setSectionError('Kindle本の取得に失敗しました')
      })
      .finally(function () {
        if (!silent) setSavedLoading(false)
      })
  }

  async function refreshSavedBooksNow() {
    const response = await listKindleBooks()
    const nextBooks = response.books ?? []
    setSavedBooks(nextBooks)
    return nextBooks
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

  const doneResults = Object.values(results)
    .filter(function (r) { return r.state === 'done' })
    .map(function (r) { return r.bookId })
    .sort()
  const doneSignature = JSON.stringify(doneResults)

  useEffect(function () {
    if (doneSignature === previousDoneSignatureRef.current) return
    previousDoneSignatureRef.current = doneSignature
    if (doneResults.length > 0) fetchSavedBooks(true)
  }, [doneSignature])

  async function syncBeforeAction(book: DisplayBook) {
    if (!extensionInstalled) {
      return undefined
    }

    return syncBook(book)
  }

  async function handleViewHighlights(book: DisplayBook) {
    setSectionError('')
    setSectionSuccess('')
    setSelectedBook(book)
    setBookHighlights([])
    setHighlightModalError('')
    setHighlightsLoading(true)

    const syncResult = await syncBeforeAction(book)
    const syncedHighlights = syncResult?.response?.highlights ?? []
    let latestSavedBooks = savedBooks
    let sourceId = getBookSourceId(book, latestSavedBooks, syncResult?.resolvedAsin)
    if (!sourceId) {
      try {
        latestSavedBooks = await refreshSavedBooksNow()
        sourceId = getBookSourceId(book, latestSavedBooks, syncResult?.resolvedAsin)
      } catch {}
    }
    const lookupMetadata = getLookupBookMetadata(book, latestSavedBooks, syncResult?.response)
    const lookupTitle = lookupMetadata.bookTitle
    const lookupAuthor = lookupMetadata.bookAuthor

    setSelectedBook(function (prev) {
      if (!prev || prev.id !== book.id) return prev
      return {
        ...prev,
        asin: prev.asin || sourceId,
        book_title: prev.book_title || lookupTitle,
        book_author: prev.book_author || lookupAuthor,
      }
    })

    if (syncResult?.state === 'error' && !sourceId) {
      if (!lookupTitle) {
        setHighlightModalError('同期に失敗しました。カードのエラー表示を確認してください')
        setHighlightsLoading(false)
        return
      }
    }
    if (!sourceId && !lookupTitle) {
      if (syncedHighlights.length > 0) {
        setBookHighlights(syncedHighlights)
        setHighlightsLoading(false)
        return
      }
      setHighlightModalError('この本の識別子を特定できませんでした')
      setHighlightsLoading(false)
      return
    }
    if (syncResult?.state === 'error') {
      setHighlightModalError('同期に失敗したため、保存済みのハイライトを表示しています')
    }

    try {
      const fetchedHighlights = await loadHighlightsForBook(sourceId, lookupTitle, lookupAuthor)
      const nextHighlights = fetchedHighlights.length > 0 ? fetchedHighlights : syncedHighlights
      setBookHighlights(nextHighlights)
      if (nextHighlights.length === 0) {
        setHighlightModalError('保存済みハイライトは見つかりませんでした')
      }
    } catch {
      if (syncedHighlights.length > 0) {
        setHighlightModalError('同期結果からハイライトを表示しています')
        setBookHighlights(syncedHighlights)
      } else {
        setHighlightModalError('ハイライト一覧の取得に失敗しました')
      }
    } finally {
      setHighlightsLoading(false)
    }
  }

  async function runGenerate(book: DisplayBook) {
    setSectionError('')
    setSectionSuccess('')
    setGeneratingBookId(book.id)
    setQuizLoading(true)
    setGenerateStatusText(function (prev) {
      return {
        ...prev,
        [book.id]: '同期状態を確認しています...',
      }
    })

    try {
      const syncResult = await syncBeforeAction(book)
      let latestSavedBooks = savedBooks
      let sourceId = getBookSourceId(book, latestSavedBooks, syncResult?.resolvedAsin)
      if (!sourceId) {
        try {
          latestSavedBooks = await refreshSavedBooksNow()
          sourceId = getBookSourceId(book, latestSavedBooks, syncResult?.resolvedAsin)
        } catch {}
      }
      if (syncResult?.state === 'error' && !sourceId) {
        setSectionError('同期に失敗しました。カードのエラー表示を確認してください')
        return
      }
      if (!sourceId) {
        setSectionError('この本の識別子を特定できませんでした')
        return
      }
      const lookupMetadata = getLookupBookMetadata(book, latestSavedBooks, syncResult?.response)
      if (syncResult?.state === 'error') {
        setSectionError('同期に失敗したため、保存済みハイライトから問題を作成します')
      }

      setGenerateStatusText(function (prev) {
        return {
          ...prev,
          [book.id]: '選択式の問題を生成しています...',
        }
      })

      setQuizBookTitle(lookupMetadata.bookTitle || book.book_title || 'Kindle 本')
      setQuizQuestions([])

      const questions = await generateQuestions('kindle_book', sourceId, {
        questionType: 'multiple_choice',
        questionCount: defaultQuestionCount,
        bookTitle: lookupMetadata.bookTitle,
        bookAuthor: lookupMetadata.bookAuthor,
      })
      if (questions.length === 0) {
        setSectionError('問題はまだ生成されませんでした。もう一度試してください')
        return
      }
      setGenerateStatusText(function (prev) {
        return {
          ...prev,
          [book.id]: `${questions.length}件の問題を表示しています...`,
        }
      })
      setQuizQuestions(questions)
      if (onQuestionsGenerated) {
        void onQuestionsGenerated(questions)
      }
      setSectionSuccess(`${questions.length}件の問題を作成しました。`)
    } catch (error) {
      setSectionError(getQuestionGenerationErrorMessage(error))
    } finally {
      setQuizLoading(false)
      setGeneratingBookId('')
      setGenerateStatusText(function (prev) {
        const next = { ...prev }
        delete next[book.id]
        return next
      })
    }
  }

  function handleGenerateClick(book: DisplayBook) {
    setSectionError('')
    setSectionSuccess('')
    void runGenerate(book)
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

  const displayBooks: DisplayBook[] = extensionInstalled && extensionBooks.length > 0
    ? mergeBooks(extensionBooks, savedBooks)
    : savedBooks.map(function (b) {
        return {
          id: b.asin,
          asin: b.asin,
          book_title: b.book_title,
          book_author: b.book_author,
          highlight_count: b.highlight_count,
        }
      })

  const isLoading = extensionInstalled ? booksLoading : savedLoading
  const displayError = booksError || sectionError

  return (
    <section style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.sm, marginBottom: theme.spacing.lg }}>
      <div style={{ border: `1px solid ${theme.colors.border}`, borderRadius: theme.radius.md, background: theme.colors.backgroundAlt, padding: theme.spacing.md }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>Kindle から作成</p>
          {extensionInstalled && (
            <button
              type="button"
              onClick={listBooksFromExtension}
              disabled={booksLoading}
              style={{
                padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
                border: `1px solid ${theme.colors.primary}`,
                borderRadius: theme.radius.sm,
                background: theme.colors.background,
                color: theme.colors.primary,
                cursor: booksLoading ? 'not-allowed' : 'pointer',
                fontSize: theme.fontSize.xs,
                opacity: booksLoading ? 0.5 : 1,
              }}
            >
              {booksLoading ? '取得中...' : 'Kindleから一覧を取得'}
            </button>
          )}
        </div>
        <p style={{ margin: `${theme.spacing.xs} 0 0`, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {extensionInstalled
            ? '一覧を見る・問題を作るを押すと、そのタイミングで最新の Kindle ハイライトを同期します。'
            : 'Kindle 同期には拡張機能が必要です。extension/ フォルダを Chrome に読み込んでください。'}
        </p>
      </div>

      {displayError && (
        <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{displayError}</p>
      )}
      {!displayError && sectionSuccess && (
        <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>{sectionSuccess}</p>
      )}
      {!displayError && booksStatus && (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>{booksStatus}</p>
      )}

      {isLoading ? (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>読み込み中...</p>
      ) : displayBooks.length === 0 ? (
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {extensionInstalled
            ? '本一覧をまだ取得していません。上のボタンを押してください。'
            : 'まだ Kindle ハイライトがありません'}
        </p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: theme.spacing.sm }}>
          {displayBooks.map(function (book) {
            const result = results[book.id]
            const generateSourceId = getBookSourceId(book, savedBooks, result?.resolvedAsin)
            const syncAvailable = Boolean(generateSourceId || book.asin)
            return (
              <KindleBookCard
                key={book.id}
                book={book}
                syncState={syncing[book.id] ?? 'idle'}
                syncStatusText={syncProgress[book.id]}
                generateStatusText={generateStatusText[book.id]}
                syncResult={result?.response}
                syncError={result?.error}
                syncAvailable={syncAvailable}
                generateEnabled={Boolean(generateSourceId)}
                isGenerating={generatingBookId === book.id}
                onViewHighlights={function () {
                  void handleViewHighlights(book)
                }}
                onGenerate={function () {
                  handleGenerateClick(book)
                }}
              />
            )
          })}
        </div>
      )}

      {selectedBook && (
        <KindleHighlightListModal
          bookTitle={selectedBook.book_title}
          highlights={bookHighlights}
          loading={highlightsLoading}
          error={highlightModalError}
          savingHighlightId={savingHighlightId}
          onClose={closeHighlightModal}
          onSaveExplanation={handleSaveExplanation}
        />
      )}
      {(quizLoading || quizQuestions.length > 0) && (
        <QuestionQuizSessionModal
          bookTitle={quizBookTitle}
          questions={quizQuestions}
          loading={quizLoading}
          sessionMode="generate"
          shareEnabled={!quizLoading && quizQuestions.length > 0}
          onShare={handleShareQuestionSet}
          onShareSuccess={function () {
            setSectionSuccess('タイムラインに投稿しました。')
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
      )}
    </section>
  )
}
