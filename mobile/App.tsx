import { StatusBar } from 'expo-status-bar'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  ActivityIndicator,
  Animated,
  AppState,
  KeyboardAvoidingView,
  Linking,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { SafeAreaProvider, SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context'
import { ShareIntentModule, useShareIntent } from 'expo-share-intent'

import {
  getCurrentUser,
  onAuthChanged,
  signInWithEmail,
  signOutUser,
  signUpWithEmail,
  type MobileAuthUser,
} from './src/api/auth'
import { getApiErrorMessage, isApiError, isApiStatus, serializeApiDebugError } from './src/api/errors'
import {
  importSharedHighlight,
  listBookHighlights,
  listBookHighlightsByMetadata,
  updateHighlightExplanation,
  type HighlightResponse,
  type ImportSharedHighlightResponse,
} from './src/api/highlights'
import { listKindleBooks, type KindleBook } from './src/api/kindle'
import {
  generateQuestions,
  listIncorrectQuestions,
  listSavedQuestions,
  manualGenerateQuestions,
  saveQuestion,
  syncQuestionStock,
  submitAnswer,
  type AnswerResult,
  type IncorrectQuestion,
  type Question,
  type QuestionStockBook,
  type SavedQuestion,
} from './src/api/questions'
import { createCheckoutSession } from './src/api/billing'
import { fetchTokenBalance, type TokenBalance } from './src/api/tokens'
import { getMe, signUpBackendUser, updateQuestionSettings, type MeResponse } from './src/api/users'
import { admobConfig, apiBaseURL, isFirebaseConfigured, mobileConfigStatus } from './src/config'
import {
  createPostComment,
  createQuestionPost,
  fetchPostQuestions,
  fetchTimeline,
  likePost,
  listPostComments,
  repostPost,
  type CreatedPost,
  type PostComment,
  type TimelinePost,
  unlikePost,
  unrepostPost,
} from './src/api/posts'
import { loadSavedHighlights, prependSavedHighlight, saveSavedHighlights } from './src/saved-highlights'
import { MobileKindleAutoSync, type MobileKindleAutoSyncStatus } from './src/kindle-sync/MobileKindleAutoSync'
import {
  buildShareIntentSignature,
  createFallbackUsername,
  draftFromShareIntent,
  emptyShareDraft,
  isShareDraftEmpty,
  mergeShareDraft,
  type ShareDraft,
} from './src/share'

type AuthMode = 'login' | 'signup'
type AppTab = 'timeline' | 'question' | 'profile'
type QuestionSource = {
  id: string
  asin: string
  bookTitle: string
  bookAuthor: string
  highlightCount: number
}
type QuizEntry = {
  question: Question
  userAnswer: string
  result: AnswerResult
}
type QuizSummaryStep = 'review' | 'share'
type QuestionListKind = 'saved' | 'incorrect' | null

export default function App() {
  const shareIntentEnabled = Platform.OS !== 'web'
  const { isReady, hasShareIntent, shareIntent, resetShareIntent, error: shareIntentError } = useShareIntent({
    disabled: !shareIntentEnabled,
    resetOnBackground: false,
    debug: __DEV__,
    scheme: 'aistudytool',
  })

  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authUser, setAuthUser] = useState<MobileAuthUser | null>(getCurrentUser())
  const [profile, setProfile] = useState<MeResponse | null>(null)
  const [profileLoading, setProfileLoading] = useState(true)
  const [authBusy, setAuthBusy] = useState(false)
  const [saveBusy, setSaveBusy] = useState(false)
  const [settingsBusy, setSettingsBusy] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [saveMessage, setSaveMessage] = useState<string | null>(null)
  const [settingsMessage, setSettingsMessage] = useState<string | null>(null)
  const [tokenBalance, setTokenBalance] = useState<TokenBalance | null>(null)
  const [tokenLoading, setTokenLoading] = useState(false)
  const [tokenMessage, setTokenMessage] = useState('')
  const [manualGenerating, setManualGenerating] = useState(false)
  const [lastSaved, setLastSaved] = useState<ImportSharedHighlightResponse | null>(null)
  const [lastShareSignature, setLastShareSignature] = useState('')
  const [activeTab, setActiveTab] = useState<AppTab>('timeline')
  const mainScrollRef = useRef<ScrollView | null>(null)

  const [savedHighlights, setSavedHighlights] = useState<HighlightResponse[]>([])
  const [syncedKindleBooks, setSyncedKindleBooks] = useState<KindleBook[]>([])
  const [syncedKindleBooksLoading, setSyncedKindleBooksLoading] = useState(false)
  const [syncedKindleBooksError, setSyncedKindleBooksError] = useState('')
  const [questionStockBooks, setQuestionStockBooks] = useState<QuestionStockBook[]>([])
  const [questionStockSyncing, setQuestionStockSyncing] = useState(false)
  const [questionStockMessage, setQuestionStockMessage] = useState('')
  const [questionStockError, setQuestionStockError] = useState('')
  const [mobileKindleSyncStatus, setMobileKindleSyncStatus] = useState<MobileKindleAutoSyncStatus>({
    state: 'idle',
    message: '',
  })
  const [timelinePosts, setTimelinePosts] = useState<TimelinePost[]>([])
  const [timelineLoading, setTimelineLoading] = useState(false)
  const [timelineLoadingMore, setTimelineLoadingMore] = useState(false)
  const [timelineError, setTimelineError] = useState('')
  const [timelineHasMore, setTimelineHasMore] = useState(true)
  const [timelineOffset, setTimelineOffset] = useState(0)
  const [timelineQuestionLoadingID, setTimelineQuestionLoadingID] = useState('')
  const [timelineQuestionErrors, setTimelineQuestionErrors] = useState<Record<string, string>>({})
  const [timelineActionErrors, setTimelineActionErrors] = useState<Record<string, string>>({})
  const [likedPostIDs, setLikedPostIDs] = useState<Record<string, boolean>>({})
  const [repostedPostIDs, setRepostedPostIDs] = useState<Record<string, boolean>>({})
  const [timelineLikeBusyID, setTimelineLikeBusyID] = useState('')
  const [timelineRepostBusyID, setTimelineRepostBusyID] = useState('')
  const [timelineDetailPostID, setTimelineDetailPostID] = useState('')

  const [savedQuestions, setSavedQuestions] = useState<SavedQuestion[]>([])
  const [savedQuestionsLoading, setSavedQuestionsLoading] = useState(false)
  const [savedQuestionsError, setSavedQuestionsError] = useState('')
  const [incorrectQuestions, setIncorrectQuestions] = useState<IncorrectQuestion[]>([])
  const [incorrectQuestionsLoading, setIncorrectQuestionsLoading] = useState(false)
  const [incorrectQuestionsError, setIncorrectQuestionsError] = useState('')
  const [questionListKind, setQuestionListKind] = useState<QuestionListKind>(null)

  const [generatedQuestions, setGeneratedQuestions] = useState<Question[]>([])
  const [generatedQuestionsLoading, setGeneratedQuestionsLoading] = useState(false)
  const [generatedQuestionsError, setGeneratedQuestionsError] = useState('')
  const [generatedQuestionSourceTitle, setGeneratedQuestionSourceTitle] = useState('')
  const [generatedQuestionBookAuthor, setGeneratedQuestionBookAuthor] = useState('')
  const [selectedQuestionSource, setSelectedQuestionSource] = useState<QuestionSource | null>(null)
  const [bookHighlights, setBookHighlights] = useState<HighlightResponse[]>([])
  const [bookHighlightsLoading, setBookHighlightsLoading] = useState(false)
  const [bookHighlightsError, setBookHighlightsError] = useState('')
  const [savingHighlightID, setSavingHighlightID] = useState('')
  const [defaultQuestionCount, setDefaultQuestionCount] = useState(3)
  const [quizOpen, setQuizOpen] = useState(false)
  const [quizCurrentIndex, setQuizCurrentIndex] = useState(0)
  const [quizSelectedOption, setQuizSelectedOption] = useState('')
  const [quizTextAnswer, setQuizTextAnswer] = useState('')
  const [quizSubmitting, setQuizSubmitting] = useState(false)
  const [quizSubmitError, setQuizSubmitError] = useState('')
  const [quizEntries, setQuizEntries] = useState<QuizEntry[]>([])
  const [quizSummaryStep, setQuizSummaryStep] = useState<QuizSummaryStep>('review')
  const [quizPostEnabled, setQuizPostEnabled] = useState(false)
  const [questionNotes, setQuestionNotes] = useState<Record<string, string>>({})
  const [savingQuestionID, setSavingQuestionID] = useState('')
  const [savedQuestionIDs, setSavedQuestionIDs] = useState<Record<string, boolean>>({})
  const [questionSaveErrors, setQuestionSaveErrors] = useState<Record<string, string>>({})
  const [quizShareBody, setQuizShareBody] = useState('')
  const [quizSharing, setQuizSharing] = useState(false)
  const [quizShareError, setQuizShareError] = useState('')
  const [quizSharedPostID, setQuizSharedPostID] = useState('')

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [username, setUsername] = useState('')
  const [draft, setDraft] = useState<ShareDraft>(emptyShareDraft())

  const storageUserKey = authUser?.uid ?? ''
  const configMissing = !mobileConfigStatus.ready
  const canSave = Boolean(profile) && !saveBusy && draft.content.trim().length > 0
  const isLoggedIn = Boolean(authUser)
  const showAuthForm = !authUser || (!profileLoading && !profile)
  const shareSignature = useMemo(
    () => buildShareIntentSignature(shareIntent),
    [shareIntent.files, shareIntent.meta, shareIntent.text, shareIntent.type, shareIntent.webUrl]
  )
  const incomingDraft = useMemo(
    () => draftFromShareIntent(shareIntent),
    [shareIntent.files, shareIntent.meta, shareIntent.text, shareIntent.type, shareIntent.webUrl]
  )
  const shareDebugSummary = [
    `受信準備: ${isReady ? 'OK' : '未完了'}`,
    `共有あり: ${hasShareIntent ? 'はい' : 'いいえ'}`,
    `受信種別: ${shareIntent.type ?? 'なし'}`,
    `本文: ${shareIntent.text?.trim() || 'なし'}`,
    `URL: ${shareIntent.webUrl?.trim() || 'なし'}`,
    `ファイル数: ${shareIntent.files?.length ?? 0}`,
    `ドラフト本文文字数: ${draft.content.trim().length}`,
    `ドラフトURL: ${draft.sourceURL.trim() || 'なし'}`,
    `今回署名: ${shareSignature || 'なし'}`,
    `前回署名: ${lastShareSignature || 'なし'}`,
    `エラー: ${shareIntentError ?? 'なし'}`,
  ].join('\n')
  const questionSources = useMemo(
    () => buildQuestionSources(syncedKindleBooks, savedHighlights),
    [savedHighlights, syncedKindleBooks]
  )
  const questionStockByBookKey = useMemo(
    () =>
      new Map(
        questionStockBooks.map((book) => [book.book_key, book] as const)
      ),
    [questionStockBooks]
  )
  const currentQuizQuestion = generatedQuestions[quizCurrentIndex]
  const isQuizSummary = quizOpen && generatedQuestions.length > 0 && quizCurrentIndex >= generatedQuestions.length
  const canShareCurrentQuiz = isLoggedIn && quizPostEnabled && quizEntries.length > 0 && generatedQuestions.length > 0
  const selectedTimelinePost = useMemo(
    () => timelinePosts.find((post) => post.id === timelineDetailPostID) ?? null,
    [timelineDetailPostID, timelinePosts]
  )

  const requestPendingShareIntent = useCallback(() => {
    if (Platform.OS !== 'ios') {
      return
    }

    console.debug('mobile.share.requestPendingShareIntent')
    ShareIntentModule?.getShareIntent('')
  }, [])

  const loadSyncedKindleBooks = useCallback(async () => {
    if (!isFirebaseConfigured() || !authUser) {
      setSyncedKindleBooks([])
      setSyncedKindleBooksLoading(false)
      setSyncedKindleBooksError('')
      return
    }

    setSyncedKindleBooksLoading(true)
    setSyncedKindleBooksError('')
    try {
      const response = await listKindleBooks()
      setSyncedKindleBooks(response.books ?? [])
    } catch (error: unknown) {
      setSyncedKindleBooksError(toReadableError(error, '同期済みKindle本の取得に失敗しました'))
    } finally {
      setSyncedKindleBooksLoading(false)
    }
  }, [authUser])

  const loadQuestionStock = useCallback(
    async (options?: { silent?: boolean }) => {
      if (!isFirebaseConfigured() || !authUser) {
        setQuestionStockBooks([])
        setQuestionStockSyncing(false)
        setQuestionStockMessage('')
        setQuestionStockError('')
        return
      }

      if (!options?.silent) {
        setQuestionStockSyncing(true)
      }
      setQuestionStockError('')

      try {
        const response = await syncQuestionStock()
        setQuestionStockBooks(response.books ?? [])

        if (response.queued_count > 0) {
          setQuestionStockMessage(`${response.queued_count}問の準備を開始しました`)
        } else if (response.skipped_due_to_daily_limit) {
          setQuestionStockMessage('今日の問題生成上限に達しています')
        } else {
          const preparingCount = (response.books ?? []).reduce((total, book) => total + Math.max(book.preparing ?? 0, 0), 0)
          setQuestionStockMessage(preparingCount > 0 ? `${preparingCount}問を準備しています` : '問題ストックは最新です')
        }
      } catch (error) {
        setQuestionStockError(toReadableError(error, '問題ストックの同期に失敗しました'))
      } finally {
        if (!options?.silent) {
          setQuestionStockSyncing(false)
        }
      }
    },
    [authUser]
  )

  const loadProfile = useCallback(async () => {
    if (!isFirebaseConfigured() || !authUser) {
      setProfile(null)
      setProfileLoading(false)
      return
    }

    setProfileLoading(true)
    try {
      const me = await getMe()
      setProfile(me)
      setDefaultQuestionCount(resolveDefaultQuestionCount(me.default_question_count))
    } catch (error) {
      if (isApiStatus(error, 404)) {
        const fallbackUsername = createFallbackUsername(authUser.email)
        await signUpBackendUser({ username: fallbackUsername })
        const me = await getMe()
        setProfile(me)
        setDefaultQuestionCount(resolveDefaultQuestionCount(me.default_question_count))
      } else {
        throw error
      }
    } finally {
      setProfileLoading(false)
    }
  }, [authUser])

  const loadTokenBalance = useCallback(async () => {
    if (!isFirebaseConfigured() || !authUser) {
      setTokenBalance(null)
      return
    }
    setTokenLoading(true)
    setTokenMessage('')
    try {
      setTokenBalance(await fetchTokenBalance())
    } catch (error) {
      setTokenMessage(toReadableError(error, 'トークン残高の取得に失敗しました'))
    } finally {
      setTokenLoading(false)
    }
  }, [authUser])

  useEffect(() => {
    if (!isFirebaseConfigured()) {
      setProfileLoading(false)
      return
    }

    const unsubscribe = onAuthChanged((user) => {
      setAuthUser(user)
      setSaveMessage(null)
      setSettingsMessage(null)
      setFormError(null)
      setLastSaved(null)
      setProfileLoading(Boolean(user))
      if (!user) {
        setProfile(null)
        setSavedHighlights([])
        setSyncedKindleBooks([])
        setSyncedKindleBooksLoading(false)
        setSyncedKindleBooksError('')
        setQuestionStockBooks([])
        setQuestionStockSyncing(false)
        setQuestionStockMessage('')
        setQuestionStockError('')
        setMobileKindleSyncStatus({ state: 'idle', message: '' })
        setTimelinePosts([])
        setTimelineError('')
        setTimelineHasMore(true)
        setTimelineOffset(0)
        setTimelineQuestionLoadingID('')
        setTimelineQuestionErrors({})
        setTimelineActionErrors({})
        setLikedPostIDs({})
        setRepostedPostIDs({})
        setTimelineLikeBusyID('')
        setTimelineRepostBusyID('')
        setTimelineDetailPostID('')
        setSavedQuestions([])
        setIncorrectQuestions([])
        setGeneratedQuestions([])
        setGeneratedQuestionSourceTitle('')
        setGeneratedQuestionBookAuthor('')
        setSelectedQuestionSource(null)
        setBookHighlights([])
        setBookHighlightsLoading(false)
        setBookHighlightsError('')
        setSavingHighlightID('')
        setQuizOpen(false)
        setQuizCurrentIndex(0)
        setQuizSelectedOption('')
        setQuizTextAnswer('')
        setQuizEntries([])
        setQuestionListKind(null)
        setQuizPostEnabled(false)
        setQuestionNotes({})
        setQuizShareBody('')
        setQuizSharing(false)
        setQuizShareError('')
        setQuizSharedPostID('')
        setDefaultQuestionCount(3)
      }
    })

    return unsubscribe
  }, [])

  useEffect(() => {
    if (!isReady) {
      return
    }

    requestPendingShareIntent()

    const appStateSubscription = AppState.addEventListener('change', (nextState) => {
      if (nextState === 'active') {
        console.debug('mobile.share.appState.active')
        requestPendingShareIntent()
        if (authUser) {
          void loadSyncedKindleBooks()
          void loadQuestionStock({ silent: true })
        }
      }
    })

    const linkingSubscription = Linking.addEventListener('url', (event) => {
      console.debug('mobile.share.url', event.url)
      requestPendingShareIntent()
    })

    return () => {
      appStateSubscription.remove()
      linkingSubscription.remove()
    }
  }, [authUser, isReady, loadQuestionStock, loadSyncedKindleBooks, requestPendingShareIntent])

  useEffect(() => {
    mainScrollRef.current?.scrollTo({ y: 0, animated: false })
  }, [activeTab])

  useEffect(() => {
    if (!authUser || !isFirebaseConfigured()) {
      setProfile(null)
      setProfileLoading(false)
      return
    }

    loadProfile().catch((error: unknown) => {
      console.debug('mobile.loadProfile.error', serializeDebugError(error))
      setProfile(null)
      setProfileLoading(false)
      setFormError(toReadableError(error, 'プロフィールの読み込みに失敗しました'))
    })
    void loadTokenBalance()
  }, [authUser, loadProfile, loadTokenBalance])

  useEffect(() => {
    if (!hasShareIntent || !shareSignature || shareSignature === lastShareSignature) {
      return
    }

    setActiveTab('profile')

    console.debug(
      'mobile.share.merge',
      JSON.stringify(
        {
          hasShareIntent,
          shareSignature,
          lastShareSignature,
          incomingDraft,
          shareIntent,
        },
        null,
        2
      )
    )

    setDraft((current) => {
      const nextDraft = mergeShareDraft(current, incomingDraft)
      console.debug(
        'mobile.share.nextDraft',
        JSON.stringify(
          {
            current,
            nextDraft,
          },
          null,
          2
        )
      )
      return nextDraft
    })

    setLastShareSignature(shareSignature)
    setSaveMessage('共有内容を読み込みました。プロフィールタブの取り込み欄から保存できます。')
    setFormError(null)
  }, [hasShareIntent, incomingDraft, lastShareSignature, shareIntent, shareSignature])

  useEffect(() => {
    if (!storageUserKey) {
      setSavedHighlights([])
      return
    }

    loadSavedHighlights(storageUserKey)
      .then((items) => setSavedHighlights(items))
      .catch(() => setSavedHighlights([]))
  }, [storageUserKey])

  useEffect(() => {
    if (!isLoggedIn) {
      setSyncedKindleBooks([])
      setSyncedKindleBooksLoading(false)
      setSyncedKindleBooksError('')
      setQuestionStockBooks([])
      setQuestionStockSyncing(false)
      setQuestionStockMessage('')
      setQuestionStockError('')
      return
    }

    void loadSyncedKindleBooks()
    void loadQuestionStock()
  }, [isLoggedIn, loadQuestionStock, loadSyncedKindleBooks])

  useEffect(() => {
    if (!isLoggedIn) {
      setTimelinePosts([])
      setTimelineError('')
      setTimelineHasMore(true)
      setTimelineOffset(0)
      return
    }

    void loadTimeline(true)
  }, [isLoggedIn])

  async function handleAuthSubmit() {
    if (!isFirebaseConfigured()) {
      return
    }

    setAuthBusy(true)
    setFormError(null)
    try {
      if (authMode === 'signup') {
        const normalizedUsername = username.trim() || createFallbackUsername(email)
        await signUpWithEmail(email.trim(), password)
        await signUpBackendUser({ username: normalizedUsername })
      } else {
        await signInWithEmail(email.trim(), password)
      }

      setEmail('')
      setPassword('')
      setUsername('')
      await loadProfile()
      setActiveTab(hasShareIntent || !isShareDraftEmpty(draft) ? 'profile' : 'timeline')
    } catch (error) {
      console.debug('mobile.handleAuthSubmit.error', serializeDebugError(error))
      setFormError(toReadableAuthError(error))
    } finally {
      setAuthBusy(false)
    }
  }

  async function handleSaveSharedHighlight() {
    if (!profile) {
      setFormError('先にログインしてください')
      return
    }
    if (!draft.content.trim()) {
      setFormError('本文が空です')
      return
    }

    setSaveBusy(true)
    setFormError(null)
    setSaveMessage(null)

    try {
      const result = await importSharedHighlight({
        content: draft.content,
        book_title: draft.bookTitle,
        book_author: draft.bookAuthor,
        source_app: draft.sourceApp,
        source_url: draft.sourceURL,
      })

      setLastSaved(result)
      if (result.saved) {
        if (result.highlight) {
          setSavedHighlights((current) => {
            const nextItems = prependSavedHighlight(current, result.highlight as HighlightResponse)
            saveSavedHighlights(storageUserKey, nextItems).catch((error: unknown) => {
              console.debug('mobile.savedHighlights.save.error', serializeDebugError(error))
            })
            return nextItems
          })
        }

        setSaveMessage('保存できました。問題はバックグラウンドで準備します。少し待ってから「問題を解く」を押してください。')
        setDraft(emptyShareDraft())
        setLastShareSignature(shareSignature)
        resetShareIntent()
        void loadQuestionStock()
      } else if (result.duplicate) {
        setSaveMessage('この共有内容はすでに保存済みです。')
        void loadQuestionStock({ silent: true })
      }
    } catch (error) {
      setFormError(toReadableError(error, '共有内容の保存に失敗しました'))
    } finally {
      setSaveBusy(false)
    }
  }

  async function handleSignOut() {
    try {
      await signOutUser()
      setProfile(null)
      setDraft(emptyShareDraft())
      setLastSaved(null)
      setSavedHighlights([])
      setSyncedKindleBooks([])
      setSyncedKindleBooksLoading(false)
      setSyncedKindleBooksError('')
      setQuestionStockBooks([])
      setQuestionStockSyncing(false)
      setQuestionStockMessage('')
      setQuestionStockError('')
      setTokenBalance(null)
      setTokenMessage('')
      setMobileKindleSyncStatus({ state: 'idle', message: '' })
      setSaveMessage(null)
      setSettingsMessage(null)
      setFormError(null)
      setQuestionListKind(null)
      setActiveTab('profile')
    } catch (error) {
      setFormError(toReadableError(error, 'ログアウトに失敗しました'))
    }
  }

  async function handleSaveQuestionSettings() {
    if (!profile) {
      setFormError('先にログインしてください')
      return
    }

    setSettingsBusy(true)
    setSettingsMessage(null)
    try {
      const updated = await updateQuestionSettings(defaultQuestionCount)
      setProfile(updated)
      setDefaultQuestionCount(resolveDefaultQuestionCount(updated.default_question_count))
      setSettingsMessage('既定の出題数を保存しました')
    } catch (error) {
      setSettingsMessage(toReadableError(error, '既定の出題数の保存に失敗しました'))
    } finally {
      setSettingsBusy(false)
    }
  }

  async function handleOpenCheckout() {
    setTokenLoading(true)
    setTokenMessage('')
    try {
      const session = await createCheckoutSession()
      await Linking.openURL(session.url)
    } catch (error) {
      setTokenMessage(toReadableError(error, 'Checkout の開始に失敗しました'))
    } finally {
      setTokenLoading(false)
    }
  }

  async function handleManualGenerateFromCurrentBook() {
    if (!selectedQuestionSource) {
      setTokenMessage('先に本のハイライト一覧を開いてください')
      return
    }
    const candidates = bookHighlights.filter((highlight) => !highlight.explanation).slice(0, 5)
    if (candidates.length < 5) {
      setTokenMessage('手動生成には問題なしハイライトが5件以上必要です')
      return
    }

    setManualGenerating(true)
    setTokenMessage('')
    try {
      const bookKey = selectedQuestionSource.asin.trim() || buildMetadataSourceID(selectedQuestionSource.bookTitle, selectedQuestionSource.bookAuthor)
      await manualGenerateQuestions(bookKey, candidates.map((highlight) => highlight.id))
      setTokenMessage('問題生成ジョブを受け付けました')
      void loadTokenBalance()
    } catch (error) {
      setTokenMessage(toReadableError(error, '問題生成の受付に失敗しました'))
    } finally {
      setManualGenerating(false)
    }
  }

  async function loadTimeline(reset = false) {
    if (!isLoggedIn) {
      return
    }

    if (reset) {
      setTimelineLoading(true)
    } else {
      setTimelineLoadingMore(true)
    }
    setTimelineError('')

    try {
      const currentOffset = reset ? 0 : timelineOffset
      const response = await fetchTimeline(20, currentOffset)
      setTimelinePosts((current) => (reset ? response.posts : [...current, ...response.posts]))
      setTimelineOffset(currentOffset + response.posts.length)
      setTimelineHasMore(response.posts.length === 20)
    } catch (error) {
      setTimelineError(toReadableError(error, 'タイムラインの取得に失敗しました'))
    } finally {
      setTimelineLoading(false)
      setTimelineLoadingMore(false)
    }
  }

  async function handleSolveTimelinePost(post: TimelinePost) {
    setTimelineDetailPostID('')
    setTimelineQuestionLoadingID(post.id)
    setTimelineQuestionErrors((current) => {
      const next = { ...current }
      delete next[post.id]
      return next
    })

    try {
      const questions = await fetchPostQuestions(post.id)
      if (questions.length === 0) {
        setTimelineQuestionErrors((current) => ({
          ...current,
          [post.id]: 'この投稿の問題はまだ取得できません',
        }))
        return
      }

      openQuizForQuestions(questions, post.book_title ?? '問題セット', {}, { openImmediately: true, postEnabled: true })
      setActiveTab('question')
    } catch (error) {
      setTimelineQuestionErrors((current) => ({
        ...current,
        [post.id]: toReadableError(error, '投稿された問題の取得に失敗しました'),
      }))
    } finally {
      setTimelineQuestionLoadingID('')
    }
  }

  function updateTimelinePost(postID: string, updater: (post: TimelinePost) => TimelinePost) {
    setTimelinePosts((current) => current.map((post) => (post.id === postID ? updater(post) : post)))
  }

  async function handleToggleLike(post: TimelinePost) {
    const isLiked = Boolean(likedPostIDs[post.id])
    setTimelineLikeBusyID(post.id)
    setTimelineActionErrors((current) => {
      const next = { ...current }
      delete next[post.id]
      return next
    })

    try {
      if (isLiked) {
        await unlikePost(post.id)
      } else {
        await likePost(post.id)
      }
      setLikedPostIDs((current) => ({
        ...current,
        [post.id]: !isLiked,
      }))
      updateTimelinePost(post.id, (current) => ({
        ...current,
        like_count: Math.max(current.like_count + (isLiked ? -1 : 1), 0),
      }))
    } catch (error) {
      setTimelineActionErrors((current) => ({
        ...current,
        [post.id]: toReadableError(error, 'いいねの更新に失敗しました'),
      }))
    } finally {
      setTimelineLikeBusyID('')
    }
  }

  async function handleToggleRepost(post: TimelinePost) {
    const isReposted = Boolean(repostedPostIDs[post.id])
    setTimelineRepostBusyID(post.id)
    setTimelineActionErrors((current) => {
      const next = { ...current }
      delete next[post.id]
      return next
    })

    try {
      if (isReposted) {
        await unrepostPost(post.id)
      } else {
        await repostPost(post.id)
      }
      setRepostedPostIDs((current) => ({
        ...current,
        [post.id]: !isReposted,
      }))
      updateTimelinePost(post.id, (current) => ({
        ...current,
        repost_count: Math.max(current.repost_count + (isReposted ? -1 : 1), 0),
      }))
    } catch (error) {
      setTimelineActionErrors((current) => ({
        ...current,
        [post.id]: toReadableError(error, 'リポストの更新に失敗しました'),
      }))
    } finally {
      setTimelineRepostBusyID('')
    }
  }

  function handleCommentCreated(postID: string) {
    updateTimelinePost(postID, (current) => ({
      ...current,
      comment_count: current.comment_count + 1,
    }))
  }

  function openQuizForQuestions(
    questions: Question[],
    title: string,
    initialNotes?: Record<string, string>,
    options?: {
      openImmediately?: boolean
      postEnabled?: boolean
    }
  ) {
    setGeneratedQuestionsError('')
    setGeneratedQuestions(questions)
    setGeneratedQuestionSourceTitle(title)
    setGeneratedQuestionBookAuthor('')
    setQuizPostEnabled(Boolean(options?.postEnabled && questions.length > 0))
    setQuestionNotes(initialNotes ?? {})
    setSavedQuestionIDs({})
    setQuestionSaveErrors({})
    setQuizEntries([])
    setQuizSummaryStep('review')
    setQuizCurrentIndex(0)
    setQuizSelectedOption('')
    setQuizTextAnswer('')
    setQuizSubmitError('')
    setQuizShareBody('')
    setQuizSharing(false)
    setQuizShareError('')
    setQuizSharedPostID('')
    setQuizOpen(Boolean(options?.openImmediately && questions.length > 0))
  }

  async function handleLoadSavedQuestions() {
    setSavedQuestionsLoading(true)
    setSavedQuestionsError('')
    try {
      const next = await listSavedQuestions()
      setSavedQuestions(next)
      if (next.length === 0) {
        setSavedQuestionsError('保存された問題がない')
      } else {
        openQuizForQuestions(
          next,
          '保存済み問題',
          next.reduce<Record<string, string>>((current, question) => {
            current[question.id] = question.note ?? ''
            return current
          }, {}),
          { openImmediately: true }
        )
      }
    } catch (error) {
      setSavedQuestionsError(toReadableError(error, '保存済み問題の取得に失敗しました'))
    } finally {
      setSavedQuestionsLoading(false)
    }
  }

  async function handleOpenSavedQuestionsList() {
    setQuestionListKind('saved')
    setSavedQuestionsLoading(true)
    setSavedQuestionsError('')
    try {
      const next = await listSavedQuestions()
      setSavedQuestions(next)
      if (next.length === 0) {
        setSavedQuestionsError('保存された問題がない')
      }
    } catch (error) {
      setSavedQuestionsError(toReadableError(error, '保存済み問題の取得に失敗しました'))
    } finally {
      setSavedQuestionsLoading(false)
    }
  }

  async function handleLoadIncorrectQuestions() {
    setIncorrectQuestionsLoading(true)
    setIncorrectQuestionsError('')
    try {
      const next = await listIncorrectQuestions()
      setIncorrectQuestions(next)
      if (next.length === 0) {
        setIncorrectQuestionsError('間違った問題がない')
      } else {
        openQuizForQuestions(
          next,
          '間違った問題',
          next.reduce<Record<string, string>>((current, question) => {
            current[question.id] = question.note ?? ''
            return current
          }, {}),
          { openImmediately: true }
        )
      }
    } catch (error) {
      setIncorrectQuestionsError(toReadableError(error, '間違った問題の取得に失敗しました'))
    } finally {
      setIncorrectQuestionsLoading(false)
    }
  }

  async function handleOpenIncorrectQuestionsList() {
    setQuestionListKind('incorrect')
    setIncorrectQuestionsLoading(true)
    setIncorrectQuestionsError('')
    try {
      const next = await listIncorrectQuestions()
      setIncorrectQuestions(next)
      if (next.length === 0) {
        setIncorrectQuestionsError('間違った問題がない')
      }
    } catch (error) {
      setIncorrectQuestionsError(toReadableError(error, '間違った問題の取得に失敗しました'))
    } finally {
      setIncorrectQuestionsLoading(false)
    }
  }

  async function handleGenerateBookQuestions(source: QuestionSource) {
    const sourceID = source.asin.trim() || buildMetadataSourceID(source.bookTitle, source.bookAuthor)

    setGeneratedQuestionsLoading(true)
    setGeneratedQuestionsError('')
    setGeneratedQuestionSourceTitle(source.bookTitle || '取り込んだ本')
    setGeneratedQuestionBookAuthor(source.bookAuthor || '')
    setQuizOpen(false)
    setQuizCurrentIndex(0)
    setQuizSelectedOption('')
    setQuizTextAnswer('')
    setQuizSubmitError('')
    setQuizEntries([])
    setQuestionNotes({})
    setSavedQuestionIDs({})
    setQuestionSaveErrors({})

    try {
      const next = await generateQuestions('kindle_book', sourceID, {
        questionCount: defaultQuestionCount,
        bookTitle: source.bookTitle,
        bookAuthor: source.bookAuthor,
      })
      openQuizForQuestions(next, source.bookTitle || '取り込んだ本', {}, {
        openImmediately: next.length > 0,
        postEnabled: next.length > 0,
      })
      setGeneratedQuestionBookAuthor(source.bookAuthor || '')
      if (next.length === 0) {
        setGeneratedQuestionsError('問題はまだ準備中です。少し待ってからもう一度試してください')
      }
    } catch (error) {
      setGeneratedQuestionsError(toReadableError(error, '問題の取得に失敗しました'))
    } finally {
      setGeneratedQuestionsLoading(false)
    }
  }

  async function handleOpenBookHighlights(source: QuestionSource) {
    setSelectedQuestionSource(source)
    setBookHighlights([])
    setBookHighlightsError('')
    setBookHighlightsLoading(true)

    try {
      const highlights = source.asin
        ? await listBookHighlights(source.asin)
        : await listBookHighlightsByMetadata(source.bookTitle, source.bookAuthor)
      setBookHighlights(highlights)
    } catch (error) {
      setBookHighlightsError(toReadableError(error, 'ハイライト一覧の取得に失敗しました'))
    } finally {
      setBookHighlightsLoading(false)
    }
  }

  async function handleSaveHighlightExplanation(highlightID: string, explanation: string) {
    setSavingHighlightID(highlightID)
    setBookHighlightsError('')
    try {
      const updated = await updateHighlightExplanation(highlightID, explanation)
      setBookHighlights((current) => current.map((highlight) => (highlight.id === highlightID ? updated : highlight)))
      setSavedHighlights((current) => {
        const nextItems = current.map((highlight) => (highlight.id === highlightID ? updated : highlight))
        saveSavedHighlights(storageUserKey, nextItems).catch((saveError: unknown) => {
          console.debug('mobile.savedHighlights.update.error', serializeDebugError(saveError))
        })
        return nextItems
      })
    } catch (error) {
      setBookHighlightsError(toReadableError(error, '解説の保存に失敗しました'))
    } finally {
      setSavingHighlightID('')
    }
  }

  function handleCloseBookHighlights() {
    setSelectedQuestionSource(null)
    setBookHighlights([])
    setBookHighlightsLoading(false)
    setBookHighlightsError('')
    setSavingHighlightID('')
  }

  function handleCloseQuizFlow() {
    setQuizOpen(false)
    setQuizCurrentIndex(0)
    setQuizSelectedOption('')
    setQuizTextAnswer('')
    setQuizSubmitError('')
    setQuizEntries([])
    setQuizSummaryStep('review')
    setQuizShareBody('')
    setQuizSharing(false)
    setQuizShareError('')
    setQuizSharedPostID('')
    setSavedQuestionIDs({})
    setQuestionSaveErrors({})
  }

  function handleStartQuiz() {
    if (generatedQuestions.length === 0) {
      return
    }

    setQuizOpen(true)
    setQuizCurrentIndex(0)
    setQuizSelectedOption('')
    setQuizTextAnswer('')
    setQuizSubmitError('')
    setQuizEntries([])
    setQuizSummaryStep('review')
    setSavedQuestionIDs({})
    setQuestionSaveErrors({})
    setQuizShareBody('')
    setQuizSharing(false)
    setQuizShareError('')
    setQuizSharedPostID('')
  }

  async function createTimelinePostFromEntries(entries: QuizEntry[]) {
    console.debug(
      'mobile.post.create.request',
      JSON.stringify(
        {
          body: quizShareBody.trim(),
          book_title: generatedQuestionSourceTitle || '問題セット',
          question_count: entries.length,
          questions: entries.map((entry, index) => ({
            question_id: entry.question.id,
            sort_order: index,
            note_length: (questionNotes[entry.question.id] ?? '').trim().length,
          })),
        },
        null,
        2
      )
    )

    const created = await createQuestionPost({
      body: quizShareBody.trim(),
      book_title: generatedQuestionSourceTitle || '問題セット',
      question_count: entries.length,
      questions: entries.map((entry, index) => ({
        question_id: entry.question.id,
        sort_order: index,
        note: (questionNotes[entry.question.id] ?? '').trim(),
      })),
      type: 'question',
    })

    console.debug('mobile.post.create.success', JSON.stringify(created, null, 2))
    setQuizSharedPostID(created.id)
    if (profile) {
      setTimelinePosts((current) => [
        buildOptimisticTimelinePost(created, profile),
        ...current.filter((post) => post.id !== created.id),
      ])
    }
    setTimelineError('')
    return created
  }

  async function handleSubmitCurrentAnswer() {
    if (!currentQuizQuestion) {
      return
    }

    const answer =
      currentQuizQuestion.question_type === 'multiple_choice' ? quizSelectedOption : quizTextAnswer.trim()
    if (!answer) {
      return
    }

    setQuizSubmitting(true)
    setQuizSubmitError('')
    try {
      const result = await submitAnswer(currentQuizQuestion.id, answer)
      setQuizEntries((current) => [...current, { question: currentQuizQuestion, userAnswer: answer, result }])
      setQuizSummaryStep('review')
      setQuizSelectedOption('')
      setQuizTextAnswer('')
      setQuizCurrentIndex((current) => current + 1)
    } catch (error) {
      setQuizSubmitError(toReadableError(error, '回答の送信に失敗しました'))
    } finally {
      setQuizSubmitting(false)
    }
  }

  async function handleSaveSolvedQuestion(questionID: string) {
    setSavingQuestionID(questionID)
    setQuestionSaveErrors((current) => {
      const next = { ...current }
      delete next[questionID]
      return next
    })

    try {
      await saveQuestion(questionID, questionNotes[questionID] ?? '')
      setSavedQuestionIDs((current) => ({
        ...current,
        [questionID]: true,
      }))
    } catch (error) {
      setQuestionSaveErrors((current) => ({
        ...current,
        [questionID]: toReadableError(error, '問題の保存に失敗しました'),
      }))
    } finally {
      setSavingQuestionID('')
    }
  }

  async function handleShareSolvedQuestionSet() {
    if (!isLoggedIn) {
      setFormError('先にログインしてください')
      setActiveTab('profile')
      return
    }

    if (quizEntries.length === 0) {
      return
    }

    if (quizSharedPostID) {
      handleCloseQuizFlow()
      setActiveTab('timeline')
      await loadTimeline(true)
      return
    }

    setQuizSharing(true)
    setQuizShareError('')

    try {
      await createTimelinePostFromEntries(quizEntries)
      handleCloseQuizFlow()
      setActiveTab('timeline')
      loadTimeline(true).catch((refreshError: unknown) => {
        console.debug('mobile.post.timelineRefresh.error', serializeDebugError(refreshError))
      })
    } catch (error) {
      console.debug('mobile.post.create.error', serializeDebugError(error))
      setQuizShareError(toReadableError(error, 'タイムラインへの投稿に失敗しました'))
    } finally {
      setQuizSharing(false)
    }
  }

  function handleResetDraft() {
    setDraft(emptyShareDraft())
    setSaveMessage(null)
    setFormError(null)
    setLastSaved(null)
    if (hasShareIntent) {
      resetShareIntent()
      setLastShareSignature(shareSignature)
    }
  }

  function handleQuestionCountDelta(delta: number) {
    setDefaultQuestionCount((current) => clampQuestionCount(current + delta))
  }

  const tabConfig = getTabConfig(activeTab)

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.safeArea} edges={['top', 'left', 'right', 'bottom']}>
        <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.keyboard}>
          <View style={styles.page}>
            <View style={styles.header}>
              <Text style={styles.eyebrow}>{tabConfig.eyebrow}</Text>
              <Text style={styles.title}>{tabConfig.title}</Text>
              <Text style={styles.subtitle}>{tabConfig.subtitle}</Text>
            </View>

            <ScrollView
              ref={mainScrollRef}
              contentContainerStyle={styles.scrollContent}
              keyboardShouldPersistTaps="handled"
            >
              {activeTab === 'timeline' ? (
                <>
                  {!isLoggedIn ? (
                    <View style={styles.card}>
                      <Text style={styles.cardTitle}>ログインするとホームが使える</Text>
                      <Text style={styles.muted}>
                        ホームではブラウザ版に近いタイムラインを見られます。共有取り込みやプロフィール設定は下のタブから進められます。
                      </Text>
                      <PrimaryButton label="プロフィールでログインする" onPress={() => setActiveTab('profile')} />
                    </View>
                  ) : (
                    <>
                      {timelineError ? (
                        <View style={styles.card}>
                          <Text style={styles.errorText}>{timelineError}</Text>
                        </View>
                      ) : null}

                      {timelineLoading && timelinePosts.length === 0 ? (
                        <View style={styles.card}>
                          <ActivityIndicator color="#000000" />
                        </View>
                      ) : null}

                      {!timelineLoading && timelinePosts.length === 0 ? (
                        <View style={styles.card}>
                          <Text style={styles.cardTitle}>まだ投稿がありません</Text>
                          <Text style={styles.muted}>問題セットが共有されると、ここに流れてきます。</Text>
                        </View>
                      ) : null}

                      {timelinePosts.map((post) => (
                        <TimelinePostCardView
                          key={post.id}
                          post={post}
                          solving={timelineQuestionLoadingID === post.id}
                          error={timelineQuestionErrors[post.id] ?? ''}
                          actionError={timelineActionErrors[post.id] ?? ''}
                          liked={Boolean(likedPostIDs[post.id])}
                          reposted={Boolean(repostedPostIDs[post.id])}
                          likeBusy={timelineLikeBusyID === post.id}
                          repostBusy={timelineRepostBusyID === post.id}
                          onSolve={() => {
                            void handleSolveTimelinePost(post)
                          }}
                          onOpen={() => {
                            setTimelineDetailPostID(post.id)
                          }}
                          onLike={() => {
                            void handleToggleLike(post)
                          }}
                          onRepost={() => {
                            void handleToggleRepost(post)
                          }}
                        />
                      ))}

                      {timelineHasMore ? (
                        <SecondaryButton
                          label={timelineLoadingMore ? '読み込み中...' : 'もっと見る'}
                          onPress={() => {
                            void loadTimeline(false)
                          }}
                          disabled={timelineLoadingMore}
                        />
                      ) : null}
                    </>
                  )}
                </>
              ) : null}

              <TimelinePostDetailModal
                visible={Boolean(selectedTimelinePost)}
                post={selectedTimelinePost}
                solving={Boolean(selectedTimelinePost && timelineQuestionLoadingID === selectedTimelinePost.id)}
                questionError={selectedTimelinePost ? timelineQuestionErrors[selectedTimelinePost.id] ?? '' : ''}
                actionError={selectedTimelinePost ? timelineActionErrors[selectedTimelinePost.id] ?? '' : ''}
                liked={Boolean(selectedTimelinePost && likedPostIDs[selectedTimelinePost.id])}
                reposted={Boolean(selectedTimelinePost && repostedPostIDs[selectedTimelinePost.id])}
                likeBusy={Boolean(selectedTimelinePost && timelineLikeBusyID === selectedTimelinePost.id)}
                repostBusy={Boolean(selectedTimelinePost && timelineRepostBusyID === selectedTimelinePost.id)}
                onClose={() => setTimelineDetailPostID('')}
                onSolve={() => {
                  if (!selectedTimelinePost) {
                    return
                  }
                  void handleSolveTimelinePost(selectedTimelinePost)
                }}
                onLike={() => {
                  if (!selectedTimelinePost) {
                    return
                  }
                  void handleToggleLike(selectedTimelinePost)
                }}
                onRepost={() => {
                  if (!selectedTimelinePost) {
                    return
                  }
                  void handleToggleRepost(selectedTimelinePost)
                }}
                onCommentCreated={handleCommentCreated}
              />

              <BookHighlightsModal
                visible={Boolean(selectedQuestionSource)}
                bookTitle={selectedQuestionSource?.bookTitle ?? ''}
                highlights={bookHighlights}
                loading={bookHighlightsLoading}
                error={bookHighlightsError}
                savingHighlightID={savingHighlightID}
                onClose={handleCloseBookHighlights}
                onSaveExplanation={handleSaveHighlightExplanation}
              />

              <QuestionHistoryListModal
                visible={questionListKind === 'saved' || questionListKind === 'incorrect'}
                title={questionListKind === 'saved' ? '保存済み問題' : '間違った問題'}
                items={questionListKind === 'saved' ? savedQuestions : incorrectQuestions}
                loading={questionListKind === 'saved' ? savedQuestionsLoading : incorrectQuestionsLoading}
                error={questionListKind === 'saved' ? savedQuestionsError : incorrectQuestionsError}
                emptyText={questionListKind === 'saved' ? '保存された問題がない' : '間違った問題がない'}
                onClose={() => setQuestionListKind(null)}
              />

              <QuizFlowModal
                visible={quizOpen && generatedQuestions.length > 0}
                totalCount={generatedQuestions.length}
                currentIndex={quizCurrentIndex}
                currentQuestion={currentQuizQuestion}
                isSummary={isQuizSummary}
                summaryStep={quizSummaryStep}
                entries={quizEntries}
                selectedOption={quizSelectedOption}
                textAnswer={quizTextAnswer}
                submitError={quizSubmitError}
                submitting={quizSubmitting}
                questionNotes={questionNotes}
                savingQuestionID={savingQuestionID}
                savedQuestionIDs={savedQuestionIDs}
                questionSaveErrors={questionSaveErrors}
                canShare={canShareCurrentQuiz}
                quizShareBody={quizShareBody}
                quizShareError={quizShareError}
                quizSharing={quizSharing}
                onSelectOption={setQuizSelectedOption}
                onChangeTextAnswer={setQuizTextAnswer}
                onSubmitAnswer={() => {
                  void handleSubmitCurrentAnswer()
                }}
                onChangeQuestionNote={(questionID, value) => {
                  setQuestionNotes((current) => ({
                    ...current,
                    [questionID]: value,
                  }))
                  setSavedQuestionIDs((current) => ({
                    ...current,
                    [questionID]: false,
                  }))
                }}
                onSaveQuestion={(questionID) => {
                  void handleSaveSolvedQuestion(questionID)
                }}
                onGoToShare={() => {
                  setQuizSummaryStep('share')
                }}
                onBackToReview={() => {
                  setQuizSummaryStep('review')
                }}
                onChangeShareBody={(value) => {
                  setQuizShareBody(value)
                  if (quizShareError) {
                    setQuizShareError('')
                  }
                }}
                onShare={() => {
                  void handleShareSolvedQuestionSet()
                }}
              />

              {activeTab === 'question' ? (
                <>
                  <View style={styles.card}>
                    <View style={styles.sectionHeaderRow}>
                      <Text style={styles.cardTitle}>問題</Text>
                      <View style={styles.badge}>
                        <Text style={styles.badgeText}>{formatQuestionCountLabel(defaultQuestionCount)}</Text>
                      </View>
                    </View>
                    <Text style={styles.muted}>
                      取り込んだ本ごとに、設定した問題数の分だけそのまま解いていけます。
                    </Text>
                  </View>

                  {!isLoggedIn ? (
                    <View style={styles.card}>
                      <Text style={styles.cardTitle}>問題を解くにはログインが必要</Text>
                      <Text style={styles.muted}>
                        Kindle 共有の取り込みはプロフィールタブにあります。学習を始めるには先にログインしてください。
                      </Text>
                      <PrimaryButton label="プロフィールを開く" onPress={() => setActiveTab('profile')} />
                    </View>
                  ) : (
                    <>
                      <View style={styles.card}>
                        <View style={styles.sectionHeaderRow}>
                          <View style={styles.flexColumn}>
                            <Text style={styles.cardTitle}>取り込んだ本から学習</Text>
                            <Text style={styles.helper}>
                              ブラウザ版で自動同期した Kindle 本も、アプリ起動時にここへ読み込みます。
                            </Text>
                            {questionStockError ? <Text style={styles.errorText}>{questionStockError}</Text> : null}
                            {!questionStockError && questionStockMessage ? (
                              <Text style={styles.helper}>{questionStockMessage}</Text>
                            ) : null}
                            {mobileKindleSyncStatus.message ? (
                              <Text
                                style={
                                  mobileKindleSyncStatus.state === 'error' || mobileKindleSyncStatus.state === 'auth_required'
                                    ? styles.errorText
                                    : mobileKindleSyncStatus.state === 'done'
                                      ? styles.successText
                                      : styles.helper
                                }
                              >
                                {mobileKindleSyncStatus.message}
                              </Text>
                            ) : null}
                          </View>
                          <View style={styles.badge}>
                            <Text style={styles.badgeText}>{questionSources.length}冊</Text>
                          </View>
                        </View>

                        {syncedKindleBooksLoading ? (
                          <Text style={styles.muted}>同期済みの Kindle 本を読み込んでいます...</Text>
                        ) : null}
                        {questionStockSyncing ? <Text style={styles.muted}>問題ストックを確認しています...</Text> : null}
                        {!syncedKindleBooksLoading && syncedKindleBooksError ? (
                          <Text style={styles.errorText}>{syncedKindleBooksError}</Text>
                        ) : null}
                        {!syncedKindleBooksLoading && !syncedKindleBooksError && questionSources.length === 0 ? (
                          <Text style={styles.muted}>
                            まだ取り込んだ本がありません。ブラウザ版の自動同期か、プロフィールタブの共有取り込みから保存してください。
                          </Text>
                        ) : null}
                        {questionSources.map((book) => (
                          (() => {
                            const stockKey = buildQuestionSourceSyncKey(book.asin, book.bookTitle, book.bookAuthor)
                            const stockStatus = questionStockByBookKey.get(stockKey)
                            const isPreparing = (stockStatus?.stock ?? 0) === 0 && (stockStatus?.preparing ?? 0) > 0

                            return (
                          <KindleBookCardView
                            key={book.id}
                            book={book}
                            stock={stockStatus?.stock}
                            target={stockStatus?.target}
                            preparing={stockStatus?.preparing}
                            isPreparing={isPreparing}
                            onSolve={() => {
                              void handleGenerateBookQuestions(book)
                            }}
                            onViewHighlights={() => {
                              void handleOpenBookHighlights(book)
                            }}
                            solving={generatedQuestionsLoading && generatedQuestionSourceTitle === (book.bookTitle || '取り込んだ本')}
                            viewingHighlights={bookHighlightsLoading && selectedQuestionSource?.id === book.id}
                          />
                            )
                          })()
                        ))}
                      </View>

                      <QuestionCollectionCardView
                        title="保存済み問題"
                        description="解き終わって保存した問題を、もう一度解いたり一覧で見返せます。"
                        countLabel={savedQuestions.length > 0 ? `${savedQuestions.length}問` : '復習'}
                        solving={savedQuestionsLoading}
                        viewing={savedQuestionsLoading && questionListKind === 'saved'}
                        onSolve={() => {
                          void handleLoadSavedQuestions()
                        }}
                        onViewList={() => {
                          void handleOpenSavedQuestionsList()
                        }}
                      />

                      <QuestionCollectionCardView
                        title="間違った問題"
                        description="直近で間違えた問題を、もう一度解いたり一覧で見返せます。"
                        countLabel={incorrectQuestions.length > 0 ? `${incorrectQuestions.length}問` : '復習'}
                        solving={incorrectQuestionsLoading}
                        viewing={incorrectQuestionsLoading && questionListKind === 'incorrect'}
                        onSolve={() => {
                          void handleLoadIncorrectQuestions()
                        }}
                        onViewList={() => {
                          void handleOpenIncorrectQuestionsList()
                        }}
                      />
                    </>
                  )}

                  {generatedQuestionsError ? (
                    <View style={styles.card}>
                      <Text style={styles.errorText}>{generatedQuestionsError}</Text>
                    </View>
                  ) : null}

                  {savedQuestionsError ? (
                    <View style={styles.card}>
                      <Text style={styles.errorText}>{savedQuestionsError}</Text>
                    </View>
                  ) : null}

                  {incorrectQuestionsError ? (
                    <View style={styles.card}>
                      <Text style={styles.errorText}>{incorrectQuestionsError}</Text>
                    </View>
                  ) : null}
                </>
              ) : null}

              {activeTab === 'profile' ? (
                <>
                  {showAuthForm ? (
                    <View style={styles.card}>
                      {authUser && !profile ? (
                        <Text style={styles.muted}>
                          プロフィールを読み込めなかったため、ログイン情報を再入力してください。
                        </Text>
                      ) : null}
                      <View style={styles.segmented}>
                        <SegmentButton label="ログイン" active={authMode === 'login'} onPress={() => setAuthMode('login')} />
                        <SegmentButton label="新規登録" active={authMode === 'signup'} onPress={() => setAuthMode('signup')} />
                      </View>

                      {authMode === 'signup' ? (
                        <Field
                          label="ユーザー名"
                          value={username}
                          placeholder="@username"
                          onChangeText={setUsername}
                          autoCapitalize="none"
                        />
                      ) : null}

                      <Field
                        label="メールアドレス"
                        value={email}
                        placeholder="you@example.com"
                        onChangeText={setEmail}
                        autoCapitalize="none"
                        keyboardType="email-address"
                      />
                      <Field
                        label="パスワード"
                        value={password}
                        placeholder="6文字以上"
                        onChangeText={setPassword}
                        secureTextEntry
                      />

                      <PrimaryButton
                        label={authBusy ? '処理中...' : authMode === 'signup' ? 'アカウントを作成' : 'ログイン'}
                        disabled={authBusy || !email.trim() || !password}
                        onPress={() => {
                          void handleAuthSubmit()
                        }}
                      />
                    </View>
                  ) : (
                    <>
                      <View style={styles.card}>
                        <View style={styles.profileHeader}>
                          <AvatarChip
                            label={buildAvatarLabel(profile?.display_name ?? profile?.username ?? authUser.email ?? 'U')}
                          />
                          <View style={styles.profileTextColumn}>
                            <Text style={styles.profileName}>
                              {profile?.display_name ?? authUser.email ?? 'ログイン中'}
                            </Text>
                            <Text style={styles.profileHandle}>
                              {profileLoading
                                ? 'プロフィールを読み込み中...'
                                : profile?.username
                                  ? `@${profile.username}`
                                  : authUser.email}
                            </Text>
                          </View>
                        </View>

                        {profileLoading ? <ActivityIndicator color="#000000" /> : null}
                        {profile?.bio ? <Text style={styles.bodyText}>{profile.bio}</Text> : null}

                        <View style={styles.statsRow}>
                          <ProfileStat label="プラン" value={profile?.plan || 'free'} />
                          <ProfileStat label="既定の出題数" value={formatQuestionCountLabel(defaultQuestionCount)} />
                        </View>
                      </View>

                      <View style={styles.card}>
                        <Text style={styles.cardTitle}>トークン管理</Text>
                        <Text style={styles.muted}>
                          今日の問題生成: {tokenBalance?.free_used_today ?? 0} / {tokenBalance?.free_limit ?? 10}問
                        </Text>
                        <Text style={styles.muted}>
                          トークン残高: {tokenBalance?.available_tokens ?? 0}問分（今日の広告: {tokenBalance?.ad_views_today ?? 0} /{' '}
                          {tokenBalance?.ad_views_limit ?? 3}回）
                        </Text>
                        <Text style={styles.helper}>Rewarded Ad unit: {admobConfig.rewardedAdUnitID}</Text>
                        {tokenMessage ? (
                          <Text style={tokenMessage.includes('失敗') || tokenMessage.includes('必要') ? styles.errorText : styles.successText}>
                            {tokenMessage}
                          </Text>
                        ) : null}
                        <View style={styles.buttonRow}>
                          <PrimaryButton
                            label="広告連携準備中"
                            onPress={() => {
                              setTokenMessage('広告トークンはサーバー検証付きの広告連携後に利用できます')
                            }}
                            disabled
                          />
                          <SecondaryButton
                            label="残高更新"
                            onPress={() => {
                              void loadTokenBalance()
                            }}
                            disabled={tokenLoading}
                          />
                        </View>
                      </View>

                      <View style={styles.card}>
                        <Text style={styles.cardTitle}>問題を生成する</Text>
                        <Text style={styles.muted}>ハイライト一覧で開いている本から、問題なしの先頭5件を手動生成します。</Text>
                        <PrimaryButton
                          label={manualGenerating ? '受付中...' : '5問を生成する'}
                          onPress={() => {
                            void handleManualGenerateFromCurrentBook()
                          }}
                          disabled={manualGenerating}
                        />
                      </View>

                      <View style={styles.card}>
                        <Text style={styles.cardTitle}>プレミアムプラン</Text>
                        <Text style={styles.muted}>月額 ¥600 で生成回数無制限</Text>
                        <PrimaryButton
                          label={tokenLoading ? '準備中...' : 'プランを確認する'}
                          onPress={() => {
                            void handleOpenCheckout()
                          }}
                          disabled={tokenLoading}
                        />
                      </View>

                      <View style={styles.card}>
                        <Text style={styles.cardTitle}>既定の出題数</Text>
                        <Text style={styles.muted}>「問題を解く」を押した時は、この設定がそのまま使われます。</Text>

                        <View style={styles.stepperRow}>
                          <SmallButton label="-" onPress={() => handleQuestionCountDelta(-1)} disabled={settingsBusy} />
                          <View style={styles.stepperValue}>
                            <Text style={styles.stepperValueText}>{formatQuestionCountLabel(defaultQuestionCount)}</Text>
                          </View>
                          <SmallButton label="+" onPress={() => handleQuestionCountDelta(1)} disabled={settingsBusy} />
                        </View>

                        {settingsMessage ? (
                          <Text style={settingsMessage.includes('失敗') ? styles.errorText : styles.successText}>
                            {settingsMessage}
                          </Text>
                        ) : null}

                        <PrimaryButton
                          label={settingsBusy ? '保存中...' : '保存'}
                          onPress={() => {
                            void handleSaveQuestionSettings()
                          }}
                          disabled={settingsBusy}
                        />
                      </View>

                      <View style={styles.card}>
                        <Text style={styles.cardTitle}>接続先</Text>
                        <Text style={styles.muted}>{apiBaseURL}</Text>
                        <Text style={styles.helper}>共有シートは Expo Go ではなく custom dev client で確認します。</Text>
                      </View>

                      <SecondaryButton label="ログアウト" onPress={() => void handleSignOut()} />
                    </>
                  )}

                  {configMissing ? (
                    <View style={styles.card}>
                      <Text style={styles.cardTitle}>設定が足りない</Text>
                      <Text style={styles.muted}>
                        {mobileConfigStatus.missing.join(', ')} を `mobile/.env` に入れるとログインと保存が動きます。
                      </Text>
                    </View>
                  ) : null}

                  <View style={styles.card}>
                    <Text style={styles.cardTitle}>モバイル共有から取り込み</Text>
                    <Text style={styles.muted}>
                      Kindle の共有シートから来た内容はここに入ります。必要なら本文や書誌情報を直してから保存できます。
                    </Text>
                    {!isLoggedIn ? <Text style={styles.helper}>保存するには先にログインしてください。</Text> : null}
                    <Field
                      label="本のタイトル"
                      value={draft.bookTitle}
                      placeholder="本のタイトル"
                      onChangeText={(value) => setDraft((current) => ({ ...current, bookTitle: value }))}
                    />
                    <Field
                      label="著者"
                      value={draft.bookAuthor}
                      placeholder="著者名"
                      onChangeText={(value) => setDraft((current) => ({ ...current, bookAuthor: value }))}
                    />
                    <Field
                      label="ハイライト本文"
                      value={draft.content}
                      placeholder="共有シートから取り込んだ本文"
                      onChangeText={(value) => setDraft((current) => ({ ...current, content: value }))}
                      multiline
                      minHeight={180}
                    />
                    <Field
                      label="共有元アプリ"
                      value={draft.sourceApp}
                      placeholder="kindle"
                      onChangeText={(value) => setDraft((current) => ({ ...current, sourceApp: value }))}
                      autoCapitalize="none"
                    />
                    <Field
                      label="共有URL"
                      value={draft.sourceURL}
                      placeholder="https://..."
                      onChangeText={(value) => setDraft((current) => ({ ...current, sourceURL: value }))}
                      autoCapitalize="none"
                      keyboardType="url"
                    />
                    <View style={styles.buttonRow}>
                      <PrimaryButton
                        label={saveBusy ? '保存中...' : '取り込みを保存'}
                        onPress={() => {
                          void handleSaveSharedHighlight()
                        }}
                        disabled={!canSave}
                      />
                      <SecondaryButton label="クリア" onPress={handleResetDraft} disabled={saveBusy || isShareDraftEmpty(draft)} />
                    </View>
                  </View>

                  {__DEV__ ? (
                    <View style={styles.card}>
                      <View style={styles.sectionHeaderRow}>
                        <View>
                          <Text style={styles.cardTitle}>共有デバッグ</Text>
                          <Text style={styles.muted}>Kindle共有で反映されない時の受信状態</Text>
                        </View>
                        <IconButton label="再取得" onPress={requestPendingShareIntent} />
                      </View>
                      <Text style={styles.debugText}>{shareDebugSummary}</Text>
                    </View>
                  ) : null}
                </>
              ) : null}

              {shareIntentError ? (
                <View style={styles.card}>
                  <Text style={styles.errorText}>{shareIntentError}</Text>
                </View>
              ) : null}

              {formError ? (
                <View style={styles.card}>
                  <Text style={styles.errorText}>{formError}</Text>
                </View>
              ) : null}

              {saveMessage ? (
                <View style={styles.card}>
                  <Text style={styles.successText}>{saveMessage}</Text>
                </View>
              ) : null}
            </ScrollView>

            <View style={styles.tabBar}>
              <TabButton label="ホーム" active={activeTab === 'timeline'} onPress={() => setActiveTab('timeline')} />
              <TabButton label="問題" active={activeTab === 'question'} onPress={() => setActiveTab('question')} />
              <TabButton label="プロフィール" active={activeTab === 'profile'} onPress={() => setActiveTab('profile')} />
            </View>
          </View>

          <StatusBar style="dark" />
          <MobileKindleAutoSync
            enabled={isLoggedIn}
            onImported={() => {
              void loadSyncedKindleBooks()
              void loadQuestionStock({ silent: true })
            }}
            onStatusChange={setMobileKindleSyncStatus}
          />
        </KeyboardAvoidingView>
      </SafeAreaView>
    </SafeAreaProvider>
  )
}

type FieldProps = {
  label: string
  value: string
  placeholder?: string
  autoCapitalize?: 'none' | 'sentences' | 'words' | 'characters'
  keyboardType?: 'default' | 'email-address' | 'url'
  secureTextEntry?: boolean
  multiline?: boolean
  minHeight?: number
  autoGrow?: boolean
  onChangeText: (value: string) => void
}

function Field({
  label,
  value,
  placeholder,
  autoCapitalize = 'sentences',
  keyboardType = 'default',
  secureTextEntry = false,
  multiline = false,
  minHeight,
  autoGrow = false,
  onChangeText,
}: FieldProps) {
  const baseHeight = minHeight ?? (multiline ? 120 : 48)
  const [contentHeight, setContentHeight] = useState(baseHeight)

  return (
    <View style={styles.fieldGroup}>
      <Text style={styles.fieldLabel}>{label}</Text>
      <TextInput
        style={[
          styles.input,
          multiline ? styles.multilineInput : null,
          { minHeight: baseHeight },
          autoGrow && multiline ? { height: Math.max(baseHeight, contentHeight) } : null,
        ]}
        value={value}
        placeholder={placeholder}
        placeholderTextColor="#536471"
        autoCapitalize={autoCapitalize}
        keyboardType={keyboardType}
        secureTextEntry={secureTextEntry}
        multiline={multiline}
        textAlignVertical={multiline ? 'top' : 'center'}
        onContentSizeChange={
          autoGrow && multiline
            ? (event) => {
                setContentHeight(event.nativeEvent.contentSize.height)
              }
            : undefined
        }
        onChangeText={onChangeText}
      />
    </View>
  )
}

function PrimaryButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.primaryButton,
        disabled ? styles.primaryButtonDisabled : null,
        pressed && !disabled ? styles.primaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={styles.primaryButtonText}>{label}</Text>
    </Pressable>
  )
}

function SecondaryButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.secondaryButton,
        disabled ? styles.secondaryButtonDisabled : null,
        pressed && !disabled ? styles.secondaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={[styles.secondaryButtonText, disabled ? styles.secondaryButtonTextDisabled : null]}>{label}</Text>
    </Pressable>
  )
}

function SegmentButton({
  label,
  active,
  onPress,
}: {
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <Pressable accessibilityRole="button" style={[styles.segmentButton, active ? styles.segmentButtonActive : null]} onPress={onPress}>
      <Text style={[styles.segmentButtonText, active ? styles.segmentButtonTextActive : null]}>{label}</Text>
    </Pressable>
  )
}

function TabButton({
  label,
  active,
  onPress,
}: {
  label: string
  active: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.tabButton,
        active ? styles.tabButtonActive : null,
        pressed ? styles.tabButtonPressed : null,
      ]}
      onPress={onPress}
    >
      <Text style={[styles.tabButtonText, active ? styles.tabButtonTextActive : null]}>{label}</Text>
    </Pressable>
  )
}

function SmallButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.smallButton,
        disabled ? styles.secondaryButtonDisabled : null,
        pressed && !disabled ? styles.secondaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={styles.smallButtonText}>{label}</Text>
    </Pressable>
  )
}

function IconButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" style={styles.iconButton} onPress={onPress}>
      <Text style={styles.iconButtonText}>{label}</Text>
    </Pressable>
  )
}

function AvatarChip({ label }: { label: string }) {
  return (
    <View style={styles.avatarChip}>
      <Text style={styles.avatarChipText}>{label}</Text>
    </View>
  )
}

function ProfileStat({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.profileStat}>
      <Text style={styles.profileStatValue}>{value}</Text>
      <Text style={styles.profileStatLabel}>{label}</Text>
    </View>
  )
}

function TimelinePostCardView({
  post,
  solving,
  error,
  actionError,
  liked,
  reposted,
  likeBusy,
  repostBusy,
  onSolve,
  onOpen,
  onLike,
  onRepost,
}: {
  post: TimelinePost
  solving: boolean
  error: string
  actionError: string
  liked: boolean
  reposted: boolean
  likeBusy: boolean
  repostBusy: boolean
  onSolve: () => void
  onOpen: () => void
  onLike: () => void
  onRepost: () => void
}) {
  const displayName = post.display_name || post.username
  const hasQuestionCard = post.type === 'question' && post.question_count > 0

  return (
    <View style={styles.card}>
      <Pressable
        accessibilityRole="button"
        style={({ pressed }) => [styles.timelineOpenArea, pressed ? styles.secondaryButtonPressed : null]}
        onPress={onOpen}
      >
        <View style={styles.timelineHeader}>
          <AvatarChip label={buildAvatarLabel(displayName)} />
          <View style={styles.timelineHeaderText}>
            <View style={styles.timelineMetaRow}>
              <Text style={styles.timelineName}>{displayName}</Text>
              <Text style={styles.timelineHandle}>@{post.username}</Text>
              <Text style={styles.timelineDate}>{formatDate(post.created_at)}</Text>
            </View>
            {post.body ? <Text style={styles.bodyText}>{post.body}</Text> : null}
          </View>
        </View>
      </Pressable>

      {hasQuestionCard ? (
        <View style={styles.innerCard}>
          <Pressable
            accessibilityRole="button"
            style={({ pressed }) => [styles.questionCardPressable, pressed ? styles.secondaryButtonPressed : null]}
            onPress={onOpen}
          >
            <Text style={styles.helper}>問題セット</Text>
            <Text style={styles.resultTitle}>{post.book_title || '本の題名なし'}</Text>
            <Text style={styles.resultMeta}>{post.question_count}問</Text>
            <Text style={styles.helper}>タップでコメントと詳細を見る</Text>
          </Pressable>
          <PrimaryButton label={solving ? '読み込み中...' : 'この問題を解く'} onPress={onSolve} disabled={solving} />
        </View>
      ) : null}

      {error ? <Text style={styles.errorText}>{error}</Text> : null}
      {actionError ? <Text style={styles.errorText}>{actionError}</Text> : null}

      <View style={styles.timelineActionRow}>
        <TimelineActionButton
          label={repostBusy ? '処理中...' : `リポスト ${post.repost_count}`}
          active={reposted}
          disabled={repostBusy}
          onPress={onRepost}
        />
        <TimelineActionButton
          label={likeBusy ? '処理中...' : `いいね ${post.like_count}`}
          active={liked}
          disabled={likeBusy}
          onPress={onLike}
        />
        <TimelineActionButton label={`コメント ${post.comment_count}`} active={false} onPress={onOpen} />
      </View>
    </View>
  )
}

function TimelinePostDetailModal({
  visible,
  post,
  solving,
  questionError,
  actionError,
  liked,
  reposted,
  likeBusy,
  repostBusy,
  onClose,
  onSolve,
  onLike,
  onRepost,
  onCommentCreated,
}: {
  visible: boolean
  post: TimelinePost | null
  solving: boolean
  questionError: string
  actionError: string
  liked: boolean
  reposted: boolean
  likeBusy: boolean
  repostBusy: boolean
  onClose: () => void
  onSolve: () => void
  onLike: () => void
  onRepost: () => void
  onCommentCreated: (postID: string) => void
}) {
  const [commentsLoading, setCommentsLoading] = useState(false)
  const [commentsError, setCommentsError] = useState('')
  const [comments, setComments] = useState<PostComment[]>([])
  const [commentDraft, setCommentDraft] = useState('')
  const [commentSubmitting, setCommentSubmitting] = useState(false)

  useEffect(() => {
    if (!visible || !post) {
      return
    }

    setCommentsLoading(true)
    setCommentsError('')
    listPostComments(post.id)
      .then((nextComments) => {
        setComments(nextComments)
      })
      .catch((loadError: unknown) => {
        setCommentsError(toReadableError(loadError, 'コメントの取得に失敗しました'))
      })
      .finally(() => {
        setCommentsLoading(false)
      })
  }, [post, visible])

  useEffect(() => {
    if (!visible) {
      setCommentDraft('')
      setCommentSubmitting(false)
      setCommentsError('')
      setComments([])
      setCommentsLoading(false)
    }
  }, [visible])

  async function handleCreateComment() {
    if (!post) {
      return
    }

    const trimmed = commentDraft.trim()
    if (!trimmed) {
      return
    }

    setCommentSubmitting(true)
    setCommentsError('')
    try {
      const created = await createPostComment(post.id, trimmed)
      setComments((current) => [created, ...current])
      setCommentDraft('')
      onCommentCreated(post.id)
    } catch (createError) {
      setCommentsError(toReadableError(createError, 'コメントの投稿に失敗しました'))
    } finally {
      setCommentSubmitting(false)
    }
  }

  if (!post) {
    return null
  }

  const displayName = post.display_name || post.username
  const hasQuestionCard = post.type === 'question' && post.question_count > 0

  return (
    <Modal animationType="slide" transparent visible={visible} onRequestClose={onClose}>
      <View style={styles.modalBackdrop}>
        <Pressable style={styles.modalDismissArea} onPress={onClose} />
        <View style={styles.modalSheet}>
          <View style={styles.modalHandle} />
          <View style={styles.sectionHeaderRow}>
            <Text style={styles.cardTitle}>投稿の詳細</Text>
            <IconButton label="閉じる" onPress={onClose} />
          </View>
          <ScrollView contentContainerStyle={styles.stack} keyboardShouldPersistTaps="handled">
            <View style={styles.card}>
              <View style={styles.timelineHeader}>
                <AvatarChip label={buildAvatarLabel(displayName)} />
                <View style={styles.timelineHeaderText}>
                  <View style={styles.timelineMetaRow}>
                    <Text style={styles.timelineName}>{displayName}</Text>
                    <Text style={styles.timelineHandle}>@{post.username}</Text>
                    <Text style={styles.timelineDate}>{formatDate(post.created_at)}</Text>
                  </View>
                  {post.body ? <Text style={styles.bodyText}>{post.body}</Text> : null}
                </View>
              </View>

              {hasQuestionCard ? (
                <View style={styles.innerCard}>
                  <Text style={styles.helper}>問題セット</Text>
                  <Text style={styles.resultTitle}>{post.book_title || '本の題名なし'}</Text>
                  <Text style={styles.resultMeta}>{post.question_count}問</Text>
                  <PrimaryButton label={solving ? '読み込み中...' : 'この問題を解く'} onPress={onSolve} disabled={solving} />
                </View>
              ) : null}

              {questionError ? <Text style={styles.errorText}>{questionError}</Text> : null}
              {actionError ? <Text style={styles.errorText}>{actionError}</Text> : null}

              <View style={styles.timelineActionRow}>
                <TimelineActionButton
                  label={repostBusy ? '処理中...' : `リポスト ${post.repost_count}`}
                  active={reposted}
                  disabled={repostBusy}
                  onPress={onRepost}
                />
                <TimelineActionButton
                  label={likeBusy ? '処理中...' : `いいね ${post.like_count}`}
                  active={liked}
                  disabled={likeBusy}
                  onPress={onLike}
                />
                <Text style={styles.timelineFooter}>コメント {post.comment_count}</Text>
              </View>
            </View>

            <View style={styles.card}>
              <Text style={styles.cardTitle}>コメント</Text>
              <Field
                label="コメントを書く"
                value={commentDraft}
                placeholder="返信をポストする"
                onChangeText={setCommentDraft}
                multiline
                minHeight={90}
              />
              <View style={styles.commentComposerActions}>
                <PrimaryButton
                  label={commentSubmitting ? 'コメント中...' : 'コメントする'}
                  onPress={() => {
                    void handleCreateComment()
                  }}
                  disabled={commentSubmitting || !commentDraft.trim()}
                />
              </View>
              {commentsError ? <Text style={styles.errorText}>{commentsError}</Text> : null}
              {commentsLoading ? <Text style={styles.muted}>コメントを読み込み中...</Text> : null}
              {!commentsLoading && comments.length === 0 ? <Text style={styles.muted}>まだコメントがありません</Text> : null}
              {!commentsLoading && comments.length > 0 ? (
                <View style={styles.stack}>
                  {comments.map((comment) => (
                    <View key={comment.id} style={styles.commentCard}>
                      <View style={styles.timelineHeader}>
                        <AvatarChip label={buildAvatarLabel(comment.display_name || comment.username)} />
                        <View style={styles.timelineHeaderText}>
                          <View style={styles.timelineMetaRow}>
                            <Text style={styles.timelineName}>{comment.display_name || comment.username}</Text>
                            <Text style={styles.timelineHandle}>@{comment.username}</Text>
                            <Text style={styles.timelineDate}>{formatCommentDate(comment.created_at)}</Text>
                          </View>
                          <Text style={styles.bodyText}>{comment.content}</Text>
                        </View>
                      </View>
                    </View>
                  ))}
                </View>
              ) : null}
            </View>
          </ScrollView>
        </View>
      </View>
    </Modal>
  )
}

function TimelineActionButton({
  label,
  active,
  disabled,
  onPress,
}: {
  label: string
  active: boolean
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.timelineActionButton,
        active ? styles.timelineActionButtonActive : null,
        disabled ? styles.secondaryButtonDisabled : null,
        pressed && !disabled ? styles.secondaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={[styles.timelineActionText, active ? styles.timelineActionTextActive : null]}>{label}</Text>
    </Pressable>
  )
}

function KindleBookCardView({
  book,
  stock,
  target,
  preparing = 0,
  isPreparing = false,
  solving,
  viewingHighlights,
  onSolve,
  onViewHighlights,
}: {
  book: QuestionSource
  stock?: number
  target?: number
  preparing?: number
  isPreparing?: boolean
  solving: boolean
  viewingHighlights: boolean
  onSolve: () => void
  onViewHighlights: () => void
}) {
  return (
    <View style={styles.innerCard}>
      <View style={styles.sectionHeaderRow}>
        <View style={styles.flexColumn}>
          <Text style={styles.resultTitle}>{book.bookTitle || '本の題名なし'}</Text>
          <Text style={styles.resultMeta}>{book.bookAuthor || '著者不明'}</Text>
        </View>
        <View style={styles.badge}>
          <Text style={styles.badgeText}>{book.highlightCount}件</Text>
        </View>
      </View>
      {!book.asin ? <Text style={styles.helper}>ASIN が無くても、本のタイトルと著者から問題を取り出せるようにしています。</Text> : null}
      {typeof stock === 'number' && typeof target === 'number' ? (
        <Text style={isPreparing ? styles.resultTitle : styles.helper}>
          {isPreparing ? `${preparing}問準備中` : `準備済み ${stock} / ${target} 問`}
        </Text>
      ) : null}
      <View style={styles.inlineButtonRow}>
        <View style={styles.inlineButtonCell}>
          <PrimaryButton label={solving ? '読み込み中...' : '問題を解く'} onPress={onSolve} disabled={solving || isPreparing} />
        </View>
        <View style={styles.inlineButtonCell}>
          <SecondaryButton
            label={viewingHighlights ? '読み込み中...' : 'ハイライト一覧'}
            onPress={onViewHighlights}
            disabled={viewingHighlights}
          />
        </View>
      </View>
    </View>
  )
}

function QuestionCollectionCardView({
  title,
  description,
  countLabel,
  solving,
  viewing,
  onSolve,
  onViewList,
}: {
  title: string
  description: string
  countLabel: string
  solving: boolean
  viewing: boolean
  onSolve: () => void
  onViewList: () => void
}) {
  return (
    <View style={styles.card}>
      <View style={styles.sectionHeaderRow}>
        <View style={styles.flexColumn}>
          <Text style={styles.cardTitle}>{title}</Text>
          <Text style={styles.helper}>{description}</Text>
        </View>
        <View style={styles.badge}>
          <Text style={styles.badgeText}>{countLabel}</Text>
        </View>
      </View>
      <View style={styles.inlineButtonRow}>
        <View style={styles.inlineButtonCell}>
          <PrimaryButton label={solving ? '読み込み中...' : '問題を解く'} onPress={onSolve} disabled={solving} />
        </View>
        <View style={styles.inlineButtonCell}>
          <SecondaryButton label={viewing ? '読み込み中...' : '一覧を見る'} onPress={onViewList} disabled={viewing} />
        </View>
      </View>
    </View>
  )
}

function BookHighlightsModal({
  visible,
  bookTitle,
  highlights,
  loading,
  error,
  savingHighlightID,
  onClose,
  onSaveExplanation,
}: {
  visible: boolean
  bookTitle: string
  highlights: HighlightResponse[]
  loading: boolean
  error: string
  savingHighlightID: string
  onClose: () => void
  onSaveExplanation: (highlightID: string, explanation: string) => Promise<void>
}) {
  const [drafts, setDrafts] = useState<Record<string, string>>({})

  useEffect(() => {
    if (!visible) {
      return
    }

    const nextDrafts: Record<string, string> = {}
    highlights.forEach((highlight) => {
      nextDrafts[highlight.id] = highlight.explanation ?? ''
    })
    setDrafts(nextDrafts)
  }, [highlights, visible])

  return (
    <Modal animationType="slide" transparent visible={visible} onRequestClose={onClose}>
      <View style={styles.modalBackdrop}>
        <Pressable style={styles.modalDismissArea} onPress={onClose} />
        <View style={styles.modalSheet}>
          <View style={styles.modalHandle} />
          <View style={styles.sectionHeaderRow}>
            <View style={styles.flexColumn}>
              <Text style={styles.cardTitle}>ハイライト一覧</Text>
              <Text style={styles.helper}>{bookTitle || 'Kindle 本'}</Text>
            </View>
            <IconButton label="閉じる" onPress={onClose} />
          </View>

          <ScrollView contentContainerStyle={styles.stack} keyboardShouldPersistTaps="handled">
            {loading ? <Text style={styles.muted}>ハイライトを読み込み中...</Text> : null}
            {!loading && error ? <Text style={styles.errorText}>{error}</Text> : null}
            {!loading && !error && highlights.length === 0 ? (
              <Text style={styles.muted}>この本のハイライトはまだありません</Text>
            ) : null}
            {!loading && !error && highlights.length > 0 ? (
              <View style={styles.stack}>
                {highlights.map((highlight) => {
                  const isSaving = savingHighlightID === highlight.id
                  return (
                    <View key={highlight.id} style={styles.innerCard}>
                      <Text style={styles.resultBody}>{highlight.content}</Text>
                      {highlight.location ? <Text style={styles.resultMeta}>{highlight.location}</Text> : null}
                      <Field
                        label="解説"
                        value={drafts[highlight.id] ?? ''}
                        placeholder="このハイライトに自分の解説を残せます"
                        onChangeText={(value) =>
                          setDrafts((current) => ({
                            ...current,
                            [highlight.id]: value,
                          }))
                        }
                        multiline
                        minHeight={110}
                      />
                      <PrimaryButton
                        label={isSaving ? '保存中...' : '解説を保存'}
                        onPress={() => {
                          void onSaveExplanation(highlight.id, drafts[highlight.id] ?? '')
                        }}
                        disabled={isSaving}
                      />
                    </View>
                  )
                })}
              </View>
            ) : null}
          </ScrollView>
        </View>
      </View>
    </Modal>
  )
}

function QuestionHistoryListModal({
  visible,
  title,
  items,
  loading,
  error,
  emptyText,
  onClose,
}: {
  visible: boolean
  title: string
  items: Array<SavedQuestion | IncorrectQuestion>
  loading: boolean
  error: string
  emptyText: string
  onClose: () => void
}) {
  return (
    <Modal animationType="slide" transparent visible={visible} onRequestClose={onClose}>
      <View style={styles.modalBackdrop}>
        <Pressable style={styles.modalDismissArea} onPress={onClose} />
        <View style={styles.modalSheet}>
          <View style={styles.modalHandle} />
          <View style={styles.sectionHeaderRow}>
            <Text style={styles.cardTitle}>{title}</Text>
            <IconButton label="閉じる" onPress={onClose} />
          </View>
          <ScrollView contentContainerStyle={styles.stack} keyboardShouldPersistTaps="handled">
            {loading ? <Text style={styles.muted}>読み込み中...</Text> : null}
            {!loading && error ? <Text style={styles.errorText}>{error}</Text> : null}
            {!loading && !error && items.length === 0 ? <Text style={styles.muted}>{emptyText}</Text> : null}
            {!loading && !error && items.length > 0 ? (
              <View style={styles.stack}>
                {items.map((question, index) => (
                  <QuestionHistoryCard
                    key={question.id}
                    label={`${title} ${index + 1}`}
                    content={question.content}
                    explanation={question.explanation}
                    note={question.note}
                  />
                ))}
              </View>
            ) : null}
          </ScrollView>
        </View>
      </View>
    </Modal>
  )
}

function QuizFlowModal({
  visible,
  totalCount,
  currentIndex,
  currentQuestion,
  isSummary,
  summaryStep,
  entries,
  selectedOption,
  textAnswer,
  submitError,
  submitting,
  questionNotes,
  savingQuestionID,
  savedQuestionIDs,
  questionSaveErrors,
  canShare,
  quizShareBody,
  quizShareError,
  quizSharing,
  onSelectOption,
  onChangeTextAnswer,
  onSubmitAnswer,
  onChangeQuestionNote,
  onSaveQuestion,
  onGoToShare,
  onBackToReview,
  onChangeShareBody,
  onShare,
}: {
  visible: boolean
  totalCount: number
  currentIndex: number
  currentQuestion?: Question
  isSummary: boolean
  summaryStep: QuizSummaryStep
  entries: QuizEntry[]
  selectedOption: string
  textAnswer: string
  submitError: string
  submitting: boolean
  questionNotes: Record<string, string>
  savingQuestionID: string
  savedQuestionIDs: Record<string, boolean>
  questionSaveErrors: Record<string, string>
  canShare: boolean
  quizShareBody: string
  quizShareError: string
  quizSharing: boolean
  onSelectOption: (value: string) => void
  onChangeTextAnswer: (value: string) => void
  onSubmitAnswer: () => void
  onChangeQuestionNote: (questionID: string, value: string) => void
  onSaveQuestion: (questionID: string) => void
  onGoToShare: () => void
  onBackToReview: () => void
  onChangeShareBody: (value: string) => void
  onShare: () => void
}) {
  const insets = useSafeAreaInsets()
  const slideX = useRef(new Animated.Value(32)).current

  useEffect(() => {
    if (!visible) {
      return
    }

    slideX.setValue(32)
    Animated.spring(slideX, {
      toValue: 0,
      useNativeDriver: true,
      damping: 18,
      stiffness: 220,
      mass: 0.9,
    }).start()
  }, [currentIndex, currentQuestion?.id, slideX, summaryStep, visible])

  const canMoveNext = currentQuestion
    ? currentQuestion.question_type === 'multiple_choice'
      ? Boolean(selectedOption)
      : Boolean(textAnswer.trim())
    : false

  return (
    <Modal animationType="slide" visible={visible} presentationStyle="fullScreen" onRequestClose={() => {}}>
      <SafeAreaView style={styles.quizModalSafeArea} edges={['left', 'right', 'bottom']}>
        <View style={styles.quizModalPage}>
          <View style={[styles.quizModalHeader, { paddingTop: insets.top + 12 }]}>
            <View style={styles.flexColumn}>
              <Text style={styles.eyebrow}>Study</Text>
              {!isSummary ? (
                <Text style={styles.quizModalSubtitle}>
                  問題 {Math.min(currentIndex + 1, totalCount)} / {totalCount}
                </Text>
              ) : (
                <Text style={styles.quizModalSubtitle}>{summaryStep === 'review' ? '解答まとめ' : 'ポスト'}</Text>
              )}
            </View>
          </View>

          <Animated.View
            style={[
              styles.quizAnimatedContent,
              {
                transform: [{ translateX: slideX }],
                opacity: slideX.interpolate({
                  inputRange: [0, 32],
                  outputRange: [1, 0.7],
                }),
              },
            ]}
          >
            {!isSummary && currentQuestion ? (
              <View style={styles.quizFullScreenBody}>
                <ScrollView
                  style={styles.quizQuestionScroll}
                  contentContainerStyle={styles.quizQuestionContent}
                  keyboardShouldPersistTaps="handled"
                >
                  <Text style={styles.quizQuestionText}>{currentQuestion.content}</Text>

                  {currentQuestion.question_type === 'multiple_choice' ? (
                    <View style={styles.stack}>
                      {currentQuestion.options.map((option) => {
                        const selected = selectedOption === option
                        return (
                          <Pressable
                            key={`${currentQuestion.id}-${option}`}
                            accessibilityRole="button"
                            style={[styles.choiceButton, selected ? styles.choiceButtonActive : null]}
                            onPress={() => onSelectOption(option)}
                          >
                            <Text style={[styles.choiceButtonText, selected ? styles.choiceButtonTextActive : null]}>
                              {option}
                            </Text>
                          </Pressable>
                        )
                      })}
                    </View>
                  ) : (
                    <Field
                      label="回答"
                      value={textAnswer}
                      placeholder="回答を入力してください"
                      onChangeText={onChangeTextAnswer}
                      multiline
                      minHeight={120}
                    />
                  )}

                  {submitError ? <Text style={styles.errorText}>{submitError}</Text> : null}
                </ScrollView>

                <View style={styles.quizFooter}>
                  <QuizArrowButton
                    label={submitting ? '...' : '→'}
                    disabled={submitting || !canMoveNext}
                    onPress={onSubmitAnswer}
                  />
                </View>
              </View>
            ) : null}

            {isSummary && summaryStep === 'review' ? (
              <ScrollView contentContainerStyle={styles.quizSummaryContent} keyboardShouldPersistTaps="handled">
                <View style={styles.card}>
                  <Text style={styles.cardTitle}>解答まとめ</Text>
                  <Text style={styles.muted}>各問題の答えと解説を見ながら、問題ごとのメモ保存までここで進められます。</Text>
                </View>
                <View style={styles.stack}>
                  {entries.map((entry, index) => {
                    const questionID = entry.question.id
                    const note = questionNotes[questionID] ?? ''
                    const isSaving = savingQuestionID === questionID
                    return (
                      <View key={questionID} style={styles.card}>
                        <Text style={styles.helper}>問題 {index + 1}</Text>
                        <Text style={styles.bodyStrong}>{entry.question.content}</Text>
                        <Text style={styles.muted}>あなたの回答: {entry.userAnswer}</Text>
                        <Text style={entry.result.is_correct ? styles.successText : styles.errorText}>
                          {entry.result.is_correct ? '正解' : `不正解 / 正解: ${entry.result.correct_answer}`}
                        </Text>
                        <Text style={styles.helper}>解説</Text>
                        <Text style={styles.muted}>{entry.result.explanation}</Text>
                        <Field
                          label="この問題のメモ"
                          value={note}
                          placeholder="この問題だけに残したいメモを書く"
                          onChangeText={(value) => onChangeQuestionNote(questionID, value)}
                          multiline
                          minHeight={48}
                          autoGrow
                        />
                        {questionSaveErrors[questionID] ? <Text style={styles.errorText}>{questionSaveErrors[questionID]}</Text> : null}
                        {savedQuestionIDs[questionID] && !questionSaveErrors[questionID] ? (
                          <Text style={styles.successText}>保存しました</Text>
                        ) : null}
                        <SecondaryButton
                          label={isSaving ? '保存中...' : 'この問題を保存'}
                          onPress={() => onSaveQuestion(questionID)}
                          disabled={isSaving}
                        />
                      </View>
                    )
                  })}
                </View>
                {canShare ? <PrimaryButton label="ポスト画面へ進む" onPress={onGoToShare} /> : null}
              </ScrollView>
            ) : null}

            {isSummary && summaryStep === 'share' ? (
              <ScrollView contentContainerStyle={styles.quizSummaryContent} keyboardShouldPersistTaps="handled">
                <View style={styles.card}>
                  <Text style={styles.cardTitle}>ポスト</Text>
                  <Text style={styles.muted}>
                    一言メモを付けて投稿できます。この本文が、問題セットの上に表示されます。
                  </Text>
                  <Field
                    label="一言メモ"
                    value={quizShareBody}
                    placeholder="この問題セットについて一言メモを書く"
                    onChangeText={onChangeShareBody}
                    multiline
                    minHeight={48}
                    autoGrow
                  />
                  {quizShareError ? <Text style={styles.errorText}>{quizShareError}</Text> : null}
                  <View style={styles.buttonRow}>
                    <PrimaryButton label={quizSharing ? 'ポスト中...' : 'ポスト'} onPress={onShare} disabled={quizSharing} />
                    <SecondaryButton label="解答まとめに戻る" onPress={onBackToReview} disabled={quizSharing} />
                  </View>
                </View>
              </ScrollView>
            ) : null}
          </Animated.View>
        </View>
      </SafeAreaView>
    </Modal>
  )
}

function QuizArrowButton({
  label,
  disabled,
  onPress,
}: {
  label: string
  disabled?: boolean
  onPress: () => void
}) {
  return (
    <Pressable
      accessibilityRole="button"
      style={({ pressed }) => [
        styles.quizArrowButton,
        disabled ? styles.primaryButtonDisabled : null,
        pressed && !disabled ? styles.primaryButtonPressed : null,
      ]}
      disabled={disabled}
      onPress={onPress}
    >
      <Text style={styles.quizArrowButtonText}>{label}</Text>
    </Pressable>
  )
}

function QuestionCardView({ question, index }: { question: Question; index: number }) {
  return (
    <View style={styles.innerCard}>
      <Text style={styles.helper}>問題 {index + 1}</Text>
      <Text style={styles.bodyStrong}>{question.content}</Text>
      {question.options.length > 0 ? (
        <View style={styles.stackTight}>
          {question.options.map((option) => (
            <Text key={`${question.id}-${option}`} style={styles.optionText}>
              ・{option}
            </Text>
          ))}
        </View>
      ) : null}
      <Text style={styles.helper}>正解: {question.correct_answer}</Text>
      <Text style={styles.muted}>{question.explanation}</Text>
    </View>
  )
}

function QuestionHistoryCard({
  label,
  content,
  explanation,
  note,
}: {
  label: string
  content: string
  explanation: string
  note?: string
}) {
  return (
    <View style={styles.innerCard}>
      <Text style={styles.helper}>{label}</Text>
      <Text style={styles.bodyStrong}>{content}</Text>
      {note ? <Text style={styles.muted}>メモ: {note}</Text> : null}
      <Text style={styles.helper}>解説</Text>
      <Text style={styles.muted}>{explanation}</Text>
    </View>
  )
}

function buildOptimisticTimelinePost(post: CreatedPost, profile: MeResponse): TimelinePost {
  return {
    id: post.id,
    user_id: post.user_id,
    body: post.body,
    type: post.type,
    book_title: post.book_title,
    question_count: post.question_count,
    repost_count: 0,
    like_count: 0,
    comment_count: 0,
    created_at: post.created_at,
    updated_at: post.updated_at,
    score: 0,
    username: profile.username,
    display_name: profile.display_name,
    avatar_url: profile.avatar_url,
  }
}

function toReadableAuthError(error: unknown): string {
  const code = typeof error === 'object' && error !== null && 'code' in error ? String(error.code) : ''

  if (code === 'auth/email-already-in-use') return 'このメールアドレスはすでに使われている'
  if (code === 'auth/invalid-credential' || code === 'auth/invalid-login-credentials') return 'メールアドレスかパスワードが違う'
  if (code === 'auth/invalid-email') return 'メールアドレスの形式が正しくない'
  if (code === 'auth/weak-password') return 'パスワードは6文字以上で入力してほしい'

  return toReadableError(error, '認証に失敗しました')
}

function toReadableError(error: unknown, fallback: string): string {
  const message = getApiErrorMessage(error)
  if (message) {
    if (message === 'questions are still preparing') return '問題はまだ準備中です'
    if (message === 'question generation failed') return '問題の準備に失敗しました'
    if (message === 'source text is unavailable') return 'この本からはまだ問題を作れません'
    return message
  }
  if (isApiError(error)) {
    return fallback
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}

function serializeDebugError(error: unknown) {
  const apiError = serializeApiDebugError(error)
  if (apiError) {
    return apiError
  }
  if (typeof error === 'object' && error !== null) {
    return {
      kind: 'object',
      name: 'name' in error ? String(error.name) : undefined,
      message: 'message' in error ? String(error.message) : undefined,
      code: 'code' in error ? String(error.code) : undefined,
    }
  }
  return {
    kind: typeof error,
    value: String(error),
  }
}

function getTabConfig(tab: AppTab): { eyebrow: string; title: string; subtitle: string } {
  if (tab === 'timeline') {
    return {
      eyebrow: 'Timeline',
      title: 'ホーム',
      subtitle: 'ブラウザ版のように、共有された問題セットを追いながらそのまま解けるタイムラインです。',
    }
  }
  if (tab === 'question') {
    return {
      eyebrow: 'Questions',
      title: '問題',
      subtitle: '取り込んだ本ごとに準備済みの問題を解いて、必要ならハイライト一覧から解説も足せます。',
    }
  }

  return {
    eyebrow: 'Profile',
    title: 'プロフィール',
    subtitle: 'アカウント、既定の出題数、モバイル共有からの取り込みをまとめて確認できます。',
  }
}

function resolveDefaultQuestionCount(value?: number): number {
  return typeof value === 'number' ? clampQuestionCount(value) : 3
}

function clampQuestionCount(value: number): number {
  if (value < 0) return 0
  if (value > 10) return 10
  return value
}

function formatQuestionCountLabel(value: number): string {
  return value === 0 ? '全部（最大20問）' : `${value}問`
}

function buildQuestionSourceKey(asin: string, bookTitle: string, bookAuthor: string): string {
  const normalizedTitle = bookTitle.trim().toLowerCase()
  const normalizedAuthor = bookAuthor.trim().toLowerCase()
  if (normalizedTitle) {
    return `meta:${normalizedTitle}::${normalizedAuthor}`
  }

  const normalizedASIN = asin.trim()
  if (normalizedASIN) {
    return `asin:${normalizedASIN}`
  }

  return 'unknown'
}

function buildQuestionSources(kindleBooks: KindleBook[], highlights: HighlightResponse[]): QuestionSource[] {
  const grouped = new Map<string, QuestionSource>()

  kindleBooks.forEach((book) => {
    const bookTitle = book.book_title.trim() || 'タイトル未設定'
    const bookAuthor = book.book_author.trim()
    const asin = book.asin.trim()
    const key = buildQuestionSourceKey(asin, bookTitle, bookAuthor)

    grouped.set(key, {
      id: key,
      asin,
      bookTitle,
      bookAuthor,
      highlightCount: Math.max(book.highlight_count, 0),
    })
  })

  highlights.forEach((highlight) => {
    const bookTitle = (highlight.book_title ?? '').trim() || 'タイトル未設定'
    const bookAuthor = (highlight.book_author ?? '').trim()
    const asin = (highlight.asin ?? '').trim()
    const key = buildQuestionSourceKey(asin, bookTitle, bookAuthor)
    const current = grouped.get(key)

    if (current) {
      current.highlightCount += 1
      if (!current.asin && asin) {
        current.asin = asin
      }
      if (!current.bookAuthor && bookAuthor) {
        current.bookAuthor = bookAuthor
      }
      if (!current.bookTitle && bookTitle) {
        current.bookTitle = bookTitle
      }
      return
    }

    grouped.set(key, {
      id: key,
      asin,
      bookTitle,
      bookAuthor,
      highlightCount: 1,
    })
  })

  return Array.from(grouped.values()).sort((left, right) => {
    if (right.highlightCount !== left.highlightCount) {
      return right.highlightCount - left.highlightCount
    }
    return left.bookTitle.localeCompare(right.bookTitle, 'ja')
  })
}

function buildMetadataSourceID(bookTitle: string, bookAuthor: string): string {
  const normalizedTitle = bookTitle.trim() || 'shared-book'
  const normalizedAuthor = bookAuthor.trim()
  return `metadata:${normalizedTitle}:${normalizedAuthor}`.slice(0, 200)
}

function buildQuestionSourceSyncKey(asin: string, bookTitle: string, bookAuthor: string): string {
  const normalizedASIN = asin.trim()
  if (normalizedASIN) {
    return normalizedASIN
  }

  return buildMetadataSourceID(bookTitle, bookAuthor)
}

function formatDate(value?: string): string {
  if (!value) {
    return ''
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return date.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' })
}

function formatCommentDate(value?: string): string {
  if (!value) {
    return ''
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return date.toLocaleString('ja-JP', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function buildAvatarLabel(value: string): string {
  return (value.trim().slice(0, 1) || 'U').toUpperCase()
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: '#ffffff',
  },
  keyboard: {
    flex: 1,
  },
  page: {
    flex: 1,
    backgroundColor: '#ffffff',
  },
  quizModalSafeArea: {
    flex: 1,
    backgroundColor: '#ffffff',
  },
  quizModalPage: {
    flex: 1,
    backgroundColor: '#ffffff',
  },
  quizModalHeader: {
    alignItems: 'flex-start',
    borderBottomColor: '#eff3f4',
    borderBottomWidth: 1,
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 14,
  },
  quizModalSubtitle: {
    color: '#536471',
    fontSize: 14,
    lineHeight: 20,
  },
  quizAnimatedContent: {
    flex: 1,
  },
  quizFullScreenBody: {
    flex: 1,
  },
  quizQuestionScroll: {
    flex: 1,
  },
  quizQuestionContent: {
    gap: 16,
    paddingHorizontal: 16,
    paddingTop: 24,
    paddingBottom: 24,
  },
  quizQuestionText: {
    color: '#0f1419',
    fontSize: 26,
    fontWeight: '700',
    lineHeight: 38,
  },
  quizFooter: {
    alignItems: 'flex-end',
    borderTopColor: '#eff3f4',
    borderTopWidth: 1,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 20,
  },
  quizArrowButton: {
    alignItems: 'center',
    backgroundColor: '#000000',
    borderRadius: 999,
    height: 56,
    justifyContent: 'center',
    width: 56,
  },
  quizArrowButtonText: {
    color: '#ffffff',
    fontSize: 24,
    fontWeight: '700',
    lineHeight: 26,
  },
  quizSummaryContent: {
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 24,
  },
  header: {
    borderBottomColor: '#eff3f4',
    borderBottomWidth: 1,
    gap: 4,
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 14,
  },
  eyebrow: {
    color: '#536471',
    fontSize: 12,
    fontWeight: '600',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
  },
  title: {
    color: '#000000',
    fontSize: 28,
    fontWeight: '700',
  },
  subtitle: {
    color: '#536471',
    fontSize: 14,
    lineHeight: 20,
  },
  scrollContent: {
    gap: 12,
    paddingHorizontal: 16,
    paddingTop: 16,
    paddingBottom: 20,
  },
  card: {
    backgroundColor: '#ffffff',
    borderColor: '#eff3f4',
    borderRadius: 16,
    borderWidth: 1,
    gap: 12,
    padding: 16,
  },
  timelineOpenArea: {
    gap: 12,
  },
  questionCardPressable: {
    gap: 4,
  },
  innerCard: {
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 12,
    borderWidth: 1,
    gap: 8,
    padding: 14,
  },
  cardTitle: {
    color: '#000000',
    fontSize: 18,
    fontWeight: '700',
  },
  muted: {
    color: '#536471',
    fontSize: 14,
    lineHeight: 20,
  },
  helper: {
    color: '#536471',
    fontSize: 13,
    lineHeight: 18,
  },
  bodyText: {
    color: '#0f1419',
    fontSize: 14,
    lineHeight: 22,
  },
  bodyStrong: {
    color: '#0f1419',
    fontSize: 15,
    fontWeight: '600',
    lineHeight: 22,
  },
  stack: {
    gap: 10,
  },
  stackTight: {
    gap: 4,
  },
  flexColumn: {
    flex: 1,
    gap: 2,
  },
  sectionHeaderRow: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  segmented: {
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    flexDirection: 'row',
    padding: 4,
  },
  segmentButton: {
    alignItems: 'center',
    borderRadius: 999,
    flex: 1,
    justifyContent: 'center',
    minHeight: 40,
  },
  segmentButtonActive: {
    backgroundColor: '#000000',
  },
  segmentButtonText: {
    color: '#536471',
    fontSize: 14,
    fontWeight: '600',
  },
  segmentButtonTextActive: {
    color: '#ffffff',
  },
  fieldGroup: {
    gap: 6,
  },
  fieldLabel: {
    color: '#0f1419',
    fontSize: 13,
    fontWeight: '600',
  },
  input: {
    backgroundColor: '#ffffff',
    borderColor: '#cfd9de',
    borderRadius: 12,
    borderWidth: 1,
    color: '#0f1419',
    fontSize: 15,
    minHeight: 48,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  multilineInput: {
    minHeight: 120,
  },
  primaryButton: {
    alignItems: 'center',
    backgroundColor: '#000000',
    borderRadius: 999,
    justifyContent: 'center',
    minHeight: 48,
    paddingHorizontal: 16,
  },
  primaryButtonPressed: {
    backgroundColor: '#1a1a1a',
  },
  primaryButtonDisabled: {
    backgroundColor: '#8899a6',
  },
  primaryButtonText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '700',
  },
  secondaryButton: {
    alignItems: 'center',
    backgroundColor: '#ffffff',
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    justifyContent: 'center',
    minHeight: 44,
    paddingHorizontal: 16,
  },
  secondaryButtonPressed: {
    backgroundColor: '#f7f9f9',
  },
  secondaryButtonDisabled: {
    opacity: 0.5,
  },
  secondaryButtonText: {
    color: '#0f1419',
    fontSize: 14,
    fontWeight: '600',
  },
  secondaryButtonTextDisabled: {
    color: '#536471',
  },
  smallButton: {
    alignItems: 'center',
    backgroundColor: '#ffffff',
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    height: 40,
    justifyContent: 'center',
    width: 40,
  },
  smallButtonText: {
    color: '#0f1419',
    fontSize: 18,
    fontWeight: '700',
  },
  iconButton: {
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    justifyContent: 'center',
    minHeight: 36,
    paddingHorizontal: 12,
  },
  iconButtonText: {
    color: '#0f1419',
    fontSize: 13,
    fontWeight: '600',
  },
  buttonRow: {
    gap: 10,
  },
  inlineButtonRow: {
    flexDirection: 'row',
    gap: 10,
  },
  inlineButtonCell: {
    flex: 1,
  },
  badge: {
    alignItems: 'center',
    backgroundColor: '#f7f9f9',
    borderRadius: 999,
    justifyContent: 'center',
    minHeight: 28,
    paddingHorizontal: 10,
  },
  badgeText: {
    color: '#536471',
    fontSize: 12,
    fontWeight: '700',
  },
  timelineHeader: {
    flexDirection: 'row',
    gap: 12,
  },
  timelineHeaderText: {
    flex: 1,
    gap: 2,
  },
  timelineMetaRow: {
    alignItems: 'center',
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 6,
  },
  timelineActionRow: {
    alignItems: 'center',
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  timelineName: {
    color: '#0f1419',
    fontSize: 15,
    fontWeight: '700',
  },
  timelineHandle: {
    color: '#536471',
    fontSize: 13,
  },
  timelineDate: {
    color: '#536471',
    fontSize: 12,
  },
  timelineFooter: {
    color: '#536471',
    fontSize: 13,
    fontWeight: '600',
  },
  timelineActionButton: {
    alignItems: 'center',
    backgroundColor: '#ffffff',
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    justifyContent: 'center',
    minHeight: 34,
    paddingHorizontal: 12,
  },
  timelineActionButtonActive: {
    backgroundColor: '#e8f5fd',
    borderColor: '#1d9bf0',
  },
  timelineActionText: {
    color: '#0f1419',
    fontSize: 13,
    fontWeight: '600',
  },
  timelineActionTextActive: {
    color: '#1d9bf0',
  },
  modalBackdrop: {
    backgroundColor: 'rgba(15, 20, 25, 0.45)',
    flex: 1,
    justifyContent: 'flex-end',
  },
  modalDismissArea: {
    flex: 1,
  },
  modalSheet: {
    backgroundColor: '#ffffff',
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    maxHeight: '88%',
    paddingHorizontal: 16,
    paddingTop: 12,
    paddingBottom: 20,
  },
  modalHandle: {
    alignSelf: 'center',
    backgroundColor: '#cfd9de',
    borderRadius: 999,
    height: 5,
    marginBottom: 12,
    width: 48,
  },
  commentSection: {
    borderTopColor: '#eff3f4',
    borderTopWidth: 1,
    gap: 10,
    paddingTop: 12,
  },
  commentComposerActions: {
    alignItems: 'flex-end',
  },
  commentCard: {
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 12,
    borderWidth: 1,
    gap: 8,
    padding: 12,
  },
  avatarChip: {
    alignItems: 'center',
    backgroundColor: '#000000',
    borderRadius: 999,
    height: 40,
    justifyContent: 'center',
    width: 40,
  },
  avatarChipText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '700',
  },
  profileHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
  },
  profileTextColumn: {
    flex: 1,
    gap: 2,
  },
  profileName: {
    color: '#0f1419',
    fontSize: 20,
    fontWeight: '700',
  },
  profileHandle: {
    color: '#536471',
    fontSize: 14,
  },
  statsRow: {
    flexDirection: 'row',
    gap: 10,
  },
  profileStat: {
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 12,
    borderWidth: 1,
    flex: 1,
    gap: 4,
    padding: 12,
  },
  profileStatValue: {
    color: '#0f1419',
    fontSize: 16,
    fontWeight: '700',
  },
  profileStatLabel: {
    color: '#536471',
    fontSize: 12,
  },
  stepperRow: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'center',
  },
  stepperValue: {
    alignItems: 'center',
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 999,
    borderWidth: 1,
    flex: 1,
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 12,
  },
  stepperValueText: {
    color: '#0f1419',
    fontSize: 15,
    fontWeight: '700',
  },
  optionText: {
    color: '#0f1419',
    fontSize: 14,
    lineHeight: 20,
  },
  choiceButton: {
    backgroundColor: '#ffffff',
    borderColor: '#eff3f4',
    borderRadius: 12,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  choiceButtonActive: {
    backgroundColor: '#000000',
    borderColor: '#000000',
  },
  choiceButtonText: {
    color: '#0f1419',
    fontSize: 14,
    lineHeight: 20,
  },
  choiceButtonTextActive: {
    color: '#ffffff',
  },
  resultPreview: {
    backgroundColor: '#f7f9f9',
    borderColor: '#eff3f4',
    borderRadius: 12,
    borderWidth: 1,
    gap: 8,
    padding: 12,
  },
  resultTitle: {
    color: '#0f1419',
    fontSize: 16,
    fontWeight: '700',
  },
  resultMeta: {
    color: '#536471',
    fontSize: 13,
  },
  resultUrl: {
    color: '#1d9bf0',
    fontSize: 13,
    lineHeight: 18,
  },
  resultBody: {
    color: '#0f1419',
    fontSize: 15,
    lineHeight: 22,
  },
  tabBar: {
    backgroundColor: '#ffffff',
    borderTopColor: '#eff3f4',
    borderTopWidth: 1,
    flexDirection: 'row',
    gap: 10,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  tabButton: {
    alignItems: 'center',
    borderRadius: 999,
    flex: 1,
    justifyContent: 'center',
    minHeight: 44,
  },
  tabButtonActive: {
    backgroundColor: '#f7f9f9',
  },
  tabButtonPressed: {
    opacity: 0.8,
  },
  tabButtonText: {
    color: '#536471',
    fontSize: 13,
    fontWeight: '600',
  },
  tabButtonTextActive: {
    color: '#000000',
    fontWeight: '700',
  },
  errorText: {
    color: '#f4212e',
    fontSize: 14,
    lineHeight: 20,
  },
  successText: {
    color: '#00ba7c',
    fontSize: 14,
    lineHeight: 20,
  },
  debugText: {
    color: '#536471',
    fontFamily: Platform.select({ ios: 'Menlo', android: 'monospace', default: 'monospace' }),
    fontSize: 12,
    lineHeight: 18,
  },
})
