import { useMemo, useState } from "react";
import { saveQuestion, submitAnswer } from "../../api/questions";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { AnswerResult, Question } from "../../types/question";

type QuizEntry = {
  question: Question;
  userAnswer: string;
  result: AnswerResult;
}

export type QuestionSharePayload = {
  body: string;
  bookTitle: string;
  questionCount: number;
  questions: Array<{
    questionId: string;
    sortOrder: number;
    note: string;
  }>;
}

type Props = {
  bookTitle: string;
  questions: Question[];
  initialNotes?: Record<string, string>;
  mergeInitialNotesIntoExplanation?: boolean;
  readonlyExplanationNotes?: Record<string, string>;
  mergedNoteLabel?: string;
  loading?: boolean;
  sessionMode?: "generate" | "solve";
  shareEnabled?: boolean;
  onShare?: (payload: QuestionSharePayload) => Promise<void>;
  onShareSuccess?: () => void;
  onClose: () => void;
}

export function QuestionQuizSessionModal({
  bookTitle,
  questions,
  initialNotes,
  mergeInitialNotesIntoExplanation = false,
  readonlyExplanationNotes,
  mergedNoteLabel = "メモ",
  loading = false,
  sessionMode,
  shareEnabled = false,
  onShare,
  onShareSuccess,
  onClose,
}: Props) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [selected, setSelected] = useState("");
  const [textAnswer, setTextAnswer] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [entries, setEntries] = useState<QuizEntry[]>([]);
  const [notes, setNotes] = useState<Record<string, string>>(initialNotes ?? {});
  const [savingQuestionId, setSavingQuestionId] = useState("");
  const [savedQuestionIds, setSavedQuestionIds] = useState<Record<string, boolean>>({});
  const [saveErrors, setSaveErrors] = useState<Record<string, string>>({});
  const [shareBody, setShareBody] = useState("");
  const [sharing, setSharing] = useState(false);
  const [shareError, setShareError] = useState("");
  const [shareSuccess, setShareSuccess] = useState("");

  const currentQuestion = questions[currentIndex];
  const isSummary = !loading && currentIndex >= questions.length && questions.length > 0;

  const progressText = useMemo(function () {
    return `${Math.min(currentIndex + 1, questions.length)} / ${questions.length}`;
  }, [currentIndex, questions.length]);

  async function handleSubmit() {
    if (!currentQuestion) {
      return;
    }

    const answer = currentQuestion.question_type === "multiple_choice" ? selected : textAnswer.trim();
    if (!answer) {
      return;
    }

    setSubmitting(true);
    setSubmitError("");
    try {
      const result = await submitAnswer(currentQuestion.id, answer);
      setEntries(function (prev) {
        return [
          ...prev,
          {
            question: currentQuestion,
            userAnswer: answer,
            result,
          },
        ];
      });
      setSelected("");
      setTextAnswer("");
      setCurrentIndex(function (prev) {
        return prev + 1;
      });
    } catch {
      setSubmitError("回答の送信に失敗しました");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSaveQuestion(questionId: string) {
    setSavingQuestionId(questionId);
    setSaveErrors(function (prev) {
      const next = { ...prev };
      delete next[questionId];
      return next;
    });
    try {
      await saveQuestion(questionId, notes[questionId] ?? "");
      setSavedQuestionIds(function (prev) {
        return {
          ...prev,
          [questionId]: true,
        };
      });
    } catch {
      setSaveErrors(function (prev) {
        return {
          ...prev,
          [questionId]: "問題の保存に失敗しました",
        };
      });
    } finally {
      setSavingQuestionId("");
    }
  }

  async function handleShare() {
    if (!onShare || entries.length === 0) {
      return;
    }

    setSharing(true);
    setShareError("");
    setShareSuccess("");
    try {
      await onShare({
        body: shareBody.trim(),
        bookTitle,
        questionCount: entries.length,
        questions: entries.map(function (entry, index) {
          return {
            questionId: entry.question.id,
            sortOrder: index,
            note: (notes[entry.question.id] ?? "").trim(),
          };
        }),
      });
      setShareSuccess("投稿しました");
      if (onShareSuccess) {
        onShareSuccess();
      }
    } catch {
      setShareError("投稿に失敗しました");
    } finally {
      setSharing(false);
    }
  }

  function buildExplanationText(entry: QuizEntry) {
    const userNote = notes[entry.question.id] ?? "";
    const readonlyNote = readonlyExplanationNotes?.[entry.question.id] ?? "";
    const mergedNote = readonlyNote.trim() || (mergeInitialNotesIntoExplanation ? userNote.trim() : "");

    if (!mergedNote) {
      return entry.result.explanation;
    }

    return `${entry.result.explanation}\n\n${mergedNoteLabel}: ${mergedNote}`;
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.5)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: theme.spacing.md,
        zIndex: 240,
      }}
      onClick={function (event) {
        if (event.target === event.currentTarget && !submitting && !savingQuestionId && !sharing) {
          onClose();
        }
      }}
    >
      <div
        style={{
          width: "100%",
          maxWidth: "760px",
          maxHeight: "88vh",
          overflowY: "auto",
          background: theme.colors.background,
          borderRadius: theme.radius.md,
          padding: theme.spacing.lg,
          display: "flex",
          flexDirection: "column",
          gap: theme.spacing.md,
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: theme.spacing.md }}>
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              {loading ? "問題を準備中" : isSummary ? "解答まとめ" : `問題 ${progressText}`}
            </p>
            <p style={{ margin: 0, fontSize: theme.fontSize.base, fontWeight: 700 }}>{bookTitle || "Kindle 本"}</p>
          </div>
          <Button variant="ghost" onClick={onClose} disabled={submitting || Boolean(savingQuestionId) || sharing}>
            閉じる
          </Button>
        </div>

        {loading ? (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
              選択式の問題文と選択肢を生成しています...
            </p>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              少し待つと、そのまま1問目が表示されます。
            </p>
          </div>
        ) : !isSummary && currentQuestion ? (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>{currentQuestion.content}</p>

            {currentQuestion.question_type === "multiple_choice" ? (
              <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
                {currentQuestion.options.map(function (option) {
                  const isSelected = selected === option;
                  return (
                    <button
                      key={option}
                      type="button"
                      onClick={function () {
                        setSelected(option);
                      }}
                      style={{
                        padding: theme.spacing.sm,
                        borderRadius: theme.radius.sm,
                        border: `1px solid ${isSelected ? theme.colors.primary : theme.colors.border}`,
                        background: isSelected ? theme.colors.primary : theme.colors.background,
                        color: isSelected ? theme.colors.background : theme.colors.primary,
                        cursor: "pointer",
                        textAlign: "left",
                      }}
                    >
                      {option}
                    </button>
                  );
                })}
              </div>
            ) : (
              <textarea
                value={textAnswer}
                onChange={function (event) {
                  setTextAnswer(event.target.value);
                }}
                rows={5}
                placeholder="回答を入力してください"
                style={{
                  width: "100%",
                  resize: "vertical",
                  padding: theme.spacing.sm,
                  border: `1px solid ${theme.colors.border}`,
                  borderRadius: theme.radius.sm,
                  fontSize: theme.fontSize.sm,
                  background: theme.colors.background,
                }}
              />
            )}

            {submitError && (
              <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{submitError}</p>
            )}

            <Button
              onClick={handleSubmit}
              loading={submitting}
              disabled={currentQuestion.question_type === "multiple_choice" ? !selected : !textAnswer.trim()}
              fullWidth
            >
              {currentIndex === questions.length - 1 ? "解き終える" : "次の問題へ"}
            </Button>
          </div>
        ) : isSummary ? (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            {entries.map(function (entry, index) {
              const isSaving = savingQuestionId === entry.question.id;
              const note = notes[entry.question.id] ?? "";
              const explanationText = buildExplanationText(entry);

              return (
                <div
                  key={entry.question.id}
                  style={{
                    border: `1px solid ${theme.colors.border}`,
                    borderRadius: theme.radius.md,
                    background: theme.colors.backgroundAlt,
                    padding: theme.spacing.md,
                    display: "flex",
                    flexDirection: "column",
                    gap: theme.spacing.sm,
                  }}
                >
                  <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>問題 {index + 1}</p>
                  <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>{entry.question.content}</p>
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>あなたの回答: {entry.userAnswer}</p>
                  <p
                    style={{
                      margin: 0,
                      fontSize: theme.fontSize.sm,
                      color: entry.result.is_correct ? theme.colors.success : theme.colors.danger,
                    }}
                  >
                    {entry.result.is_correct ? "正解" : `不正解 / 正解: ${entry.result.correct_answer}`}
                  </p>
                  {entry.result.feedback && (
                    <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>{entry.result.feedback}</p>
                  )}
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary, whiteSpace: "pre-wrap" }}>
                    解説: {explanationText}
                  </p>
                  {note.trim() && !mergeInitialNotesIntoExplanation && (
                    <div
                      style={{
                        borderRadius: theme.radius.sm,
                        background: theme.colors.background,
                        border: `1px solid ${theme.colors.border}`,
                        padding: theme.spacing.sm,
                        display: "flex",
                        flexDirection: "column",
                        gap: theme.spacing.xs,
                      }}
                    >
                      <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                        自分のメモ
                      </p>
                      <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>
                        {note}
                      </p>
                    </div>
                  )}

                  <textarea
                    value={note}
                    onChange={function (event) {
                      const nextValue = event.target.value;
                      setNotes(function (prev) {
                        return {
                          ...prev,
                          [entry.question.id]: nextValue,
                        };
                      });
                      setSavedQuestionIds(function (prev) {
                        if (!prev[entry.question.id]) {
                          return prev;
                        }
                        return {
                          ...prev,
                          [entry.question.id]: false,
                        };
                      });
                    }}
                    rows={4}
                    placeholder="自分の解説や補足を書けます"
                    style={{
                      width: "100%",
                      resize: "vertical",
                      padding: theme.spacing.sm,
                      border: `1px solid ${theme.colors.border}`,
                      borderRadius: theme.radius.sm,
                      fontSize: theme.fontSize.sm,
                      background: theme.colors.background,
                    }}
                  />

                  {saveErrors[entry.question.id] && (
                    <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
                      {saveErrors[entry.question.id]}
                    </p>
                  )}
                  {savedQuestionIds[entry.question.id] && !saveErrors[entry.question.id] && (
                    <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>
                      保存しました
                    </p>
                  )}

                  <div style={{ display: "flex", justifyContent: "flex-end" }}>
                    <Button
                      onClick={function () {
                        void handleSaveQuestion(entry.question.id);
                      }}
                      loading={isSaving}
                      disabled={isSaving}
                    >
                      問題を保存
                    </Button>
                  </div>
                </div>
              );
            })}

            {shareEnabled && onShare && sessionMode !== "generate" && (
              <div style={{ border: `1px solid ${theme.colors.border}`, borderRadius: theme.radius.md, background: theme.colors.backgroundAlt, padding: theme.spacing.md, display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
                <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>どうだった？</p>
                <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
                  一言つけて投稿すると、タイムラインから他の人もこの問題を解けます。
                </p>
                <textarea
                  value={shareBody}
                  onChange={function (event) {
                    setShareBody(event.target.value);
                    if (shareError) setShareError("");
                    if (shareSuccess) setShareSuccess("");
                  }}
                  rows={3}
                  maxLength={280}
                  placeholder="どうだったかを一言で書く"
                  style={{ width: "100%", resize: "vertical", padding: theme.spacing.sm, border: `1px solid ${theme.colors.border}`, borderRadius: theme.radius.sm, fontSize: theme.fontSize.sm, background: theme.colors.background }}
                />
                {shareError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{shareError}</p>}
                {shareSuccess && <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>{shareSuccess}</p>}
                <div style={{ display: "flex", justifyContent: "flex-end" }}>
                  <Button onClick={function () { void handleShare(); }} loading={sharing} disabled={sharing}>
                    投稿する
                  </Button>
                </div>
              </div>
            )}

            {shareEnabled && onShare && sessionMode === "generate" && (
              <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
                <textarea
                  value={shareBody}
                  onChange={function (event) {
                    setShareBody(event.target.value);
                    if (shareError) setShareError("");
                  }}
                  rows={3}
                  maxLength={280}
                  placeholder="一言コメント（任意）"
                  style={{ width: "100%", resize: "vertical", padding: theme.spacing.sm, border: `1px solid ${theme.colors.border}`, borderRadius: theme.radius.sm, fontSize: theme.fontSize.sm, background: theme.colors.background }}
                />
                {shareError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{shareError}</p>}
              </div>
            )}

            <Button
              onClick={function () {
                if (sessionMode === "generate" && shareEnabled && onShare && entries.length > 0) {
                  void handleShare();
                } else {
                  onClose();
                }
              }}
              loading={sharing}
              disabled={sharing}
              fullWidth
            >
              {sessionMode === "generate" && shareEnabled && onShare && entries.length > 0 ? "投稿して閉じる" : "閉じる"}
            </Button>
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              問題はまだありません
            </p>
            <Button onClick={onClose} fullWidth>
              閉じる
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
