import { useEffect, useRef, useState } from 'react'
import { Animated, Modal, Pressable, ScrollView, Text, View } from 'react-native'
import { SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context'

import { getApiErrorMessage, isApiError, serializeApiDebugError } from '../api/errors'
import type { HighlightResponse } from '../api/highlights'
import { createPostComment, listPostComments, type PostComment, type TimelinePost } from '../api/posts'
import type { AnswerResult, IncorrectQuestion, Question, SavedQuestion } from '../api/questions'
import { styles } from '../appStyles'
import { AvatarChip, Field, IconButton, PrimaryButton, SecondaryButton } from './AppPrimitives'

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

export function TimelinePostCardView({
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
          label={repostBusy ? '…' : `↻ ${post.repost_count}`}
          active={reposted}
          disabled={repostBusy}
          onPress={onRepost}
        />
        <TimelineActionButton
          label={likeBusy ? '…' : `♡ ${post.like_count}`}
          active={liked}
          disabled={likeBusy}
          onPress={onLike}
        />
        <TimelineActionButton label={`＋ ${post.comment_count}`} active={false} onPress={onOpen} />
      </View>
    </View>
  )
}

export function TimelinePostDetailModal({
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
                  label={repostBusy ? '…' : `↻ ${post.repost_count}`}
                  active={reposted}
                  disabled={repostBusy}
                  onPress={onRepost}
                />
                <TimelineActionButton
                  label={likeBusy ? '…' : `♡ ${post.like_count}`}
                  active={liked}
                  disabled={likeBusy}
                  onPress={onLike}
                />
                <Text style={styles.timelineFooter}>＋ {post.comment_count}</Text>
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

export function TimelineActionButton({
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

export function KindleBookCardView({
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

export function QuestionCollectionCardView({
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

export function BookHighlightsModal({
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

export function QuestionHistoryListModal({
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

export function QuizFlowModal({
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

export function QuizArrowButton({
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

export function QuestionCardView({ question, index }: { question: Question; index: number }) {
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

export function QuestionHistoryCard({
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


function toReadableError(error: unknown, fallback: string): string {
  if (isApiError(error)) {
    const message = getApiErrorMessage(error)
    if (message) {
      return message
    }
  }

  const debugMessage = serializeApiDebugError(error)
  if (debugMessage && __DEV__) {
    return `${fallback}: ${JSON.stringify(debugMessage)}`
  }
  return fallback
}

function formatDate(value?: string): string {
  if (!value) {
    return ''
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return date.toLocaleDateString('ja-JP', {
    month: 'short',
    day: 'numeric',
  })
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
