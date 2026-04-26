import { useEffect, useMemo, useRef, useState } from "react";
import { saveQuestion, submitAnswer } from "../../api/questions";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { AnswerResult, Question } from "../../types/question";

type QuizEntry = {
  question: Question;
  userAnswer: string;
  result: AnswerResult;
};

type SummaryStep = "review" | "share";

export type QuestionSharePayload = {
  body: string;
  bookTitle: string;
  questionCount: number;
  questions: Array<{
    questionId: string;
    sortOrder: number;
    note: string;
  }>;
};

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
};

export function QuestionQuizSessionModal({
  bookTitle,
  questions,
  initialNotes,
  mergeInitialNotesIntoExplanation = false,
  readonlyExplanationNotes,
  mergedNoteLabel = "メモ",
  loading = false,
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
  const [summaryStep, setSummaryStep] = useState<SummaryStep>("review");
  const [entered, setEntered] = useState(false);

  const currentQuestion = questions[currentIndex];
  const isSummary = !loading && currentIndex >= questions.length && questions.length > 0;

  useEffect(
    function () {
      setNotes(initialNotes ?? {});
    },
    [initialNotes, questions]
  );

  useEffect(
    function () {
      setEntered(false);
      const frameId = window.requestAnimationFrame(function () {
        setEntered(true);
      });
      return function () {
        window.cancelAnimationFrame(frameId);
      };
    },
    [currentIndex, currentQuestion?.id, isSummary, summaryStep]
  );

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
      setSummaryStep("review");
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

  const canMoveNext = currentQuestion
    ? currentQuestion.question_type === "multiple_choice"
      ? Boolean(selected)
      : Boolean(textAnswer.trim())
    : false;

  const stageStyle = {
    transform: entered ? "translateX(0)" : "translateX(24px)",
    opacity: entered ? 1 : 0.7,
    transition: "transform 220ms ease, opacity 220ms ease",
  } as const;

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: theme.colors.background,
        display: "flex",
        flexDirection: "column",
        zIndex: 240,
      }}
    >
      <div
        style={{
          borderBottom: `1px solid ${theme.colors.border}`,
          padding: `${theme.spacing.md} ${theme.spacing.lg}`,
          display: "flex",
          flexDirection: "column",
          gap: theme.spacing.xs,
        }}
      >
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm, fontWeight: 700 }}>
          Study
        </p>
        <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
          {loading ? "問題を準備中" : isSummary ? (summaryStep === "review" ? "解答まとめ" : "ポスト") : `問題 ${progressText}`}
        </p>
      </div>

      {loading ? (
        <div
          style={{
            flex: 1,
            padding: theme.spacing.xl,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.md,
            justifyContent: "center",
          }}
        >
          <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.lg }}>
            問題を準備しています...
          </p>
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            少し待つと、そのまま1問目が表示されます。
          </p>
        </div>
      ) : !isSummary && currentQuestion ? (
        <div style={{ ...stageStyle, flex: 1, display: "flex", flexDirection: "column", minHeight: 0 }}>
          <div
            style={{
              flex: 1,
              overflowY: "auto",
              padding: `${theme.spacing.xl} ${theme.spacing.lg}`,
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.lg,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: "1.75rem", lineHeight: 1.5 }}>
              {currentQuestion.content}
            </p>

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
                        padding: `${theme.spacing.md} ${theme.spacing.lg}`,
                        borderRadius: theme.radius.md,
                        border: `1px solid ${isSelected ? theme.colors.primary : theme.colors.border}`,
                        background: isSelected ? theme.colors.primary : theme.colors.background,
                        color: isSelected ? theme.colors.background : "#0f1419",
                        cursor: "pointer",
                        textAlign: "left",
                        fontSize: theme.fontSize.base,
                      }}
                    >
                      {option}
                    </button>
                  );
                })}
              </div>
            ) : (
              <AutoGrowTextarea
                value={textAnswer}
                onChange={setTextAnswer}
                placeholder="回答を入力してください"
                minHeight={120}
              />
            )}

            {submitError && (
              <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{submitError}</p>
            )}
          </div>

          <div
            style={{
              borderTop: `1px solid ${theme.colors.border}`,
              padding: `${theme.spacing.md} ${theme.spacing.lg}`,
              display: "flex",
              justifyContent: "flex-end",
            }}
          >
            <button
              type="button"
              onClick={function () {
                void handleSubmit();
              }}
              disabled={submitting || !canMoveNext}
              style={{
                width: "3.5rem",
                height: "3.5rem",
                borderRadius: theme.radius.full,
                border: "none",
                background: submitting || !canMoveNext ? theme.colors.secondary : theme.colors.primary,
                color: theme.colors.background,
                cursor: submitting || !canMoveNext ? "not-allowed" : "pointer",
                fontSize: "1.5rem",
                fontWeight: 700,
              }}
            >
              {submitting ? "..." : "→"}
            </button>
          </div>
        </div>
      ) : isSummary && summaryStep === "review" ? (
        <div
          style={{
            ...stageStyle,
            flex: 1,
            overflowY: "auto",
            padding: theme.spacing.lg,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.md,
          }}
        >
          <div
            style={{
              border: `1px solid ${theme.colors.border}`,
              borderRadius: theme.radius.md,
              background: theme.colors.background,
              padding: theme.spacing.md,
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.sm,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>解答まとめ</p>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              各問題の答えと解説を見ながら、問題ごとのメモ保存までここで進められます。
            </p>
          </div>

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
                <AutoGrowTextarea
                  value={note}
                  onChange={function (nextValue) {
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
                  placeholder="この問題だけに残したいメモを書く"
                  minHeight={44}
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
                    この問題を保存
                  </Button>
                </div>
              </div>
            );
          })}

          {shareEnabled && onShare ? (
            <Button fullWidth onClick={function () { setSummaryStep("share"); }}>
              ポスト画面へ進む
            </Button>
          ) : (
            <Button variant="outline" fullWidth onClick={onClose}>
              完了して戻る
            </Button>
          )}
        </div>
      ) : isSummary ? (
        <div
          style={{
            ...stageStyle,
            flex: 1,
            overflowY: "auto",
            padding: theme.spacing.lg,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.md,
          }}
        >
          <div
            style={{
              border: `1px solid ${theme.colors.border}`,
              borderRadius: theme.radius.md,
              background: theme.colors.background,
              padding: theme.spacing.md,
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.sm,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>ポスト</p>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              一言メモを付けて投稿できます。この本文が、問題セットの上に表示されます。
            </p>
            <AutoGrowTextarea
              value={shareBody}
              onChange={function (nextValue) {
                setShareBody(nextValue);
                if (shareError) setShareError("");
                if (shareSuccess) setShareSuccess("");
              }}
              placeholder="この問題セットについて一言メモを書く"
              minHeight={44}
            />
            {shareError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{shareError}</p>}
            {shareSuccess && <p style={{ margin: 0, color: theme.colors.success, fontSize: theme.fontSize.sm }}>{shareSuccess}</p>}
            <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
              <Button fullWidth onClick={function () { void handleShare(); }} loading={sharing} disabled={sharing}>
                ポスト
              </Button>
              <Button variant="outline" fullWidth onClick={function () { setSummaryStep("review"); }} disabled={sharing}>
                解答まとめに戻る
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <div
          style={{
            flex: 1,
            padding: theme.spacing.lg,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.md,
            justifyContent: "center",
          }}
        >
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            問題はまだありません
          </p>
          <Button variant="outline" fullWidth onClick={onClose}>
            戻る
          </Button>
        </div>
      )}
    </div>
  );
}

function AutoGrowTextarea({
  value,
  onChange,
  placeholder,
  minHeight = 44,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  minHeight?: number;
}) {
  const ref = useRef<HTMLTextAreaElement | null>(null);

  useEffect(
    function () {
      if (!ref.current) {
        return;
      }
      ref.current.style.height = "auto";
      ref.current.style.height = `${Math.max(ref.current.scrollHeight, minHeight)}px`;
    },
    [minHeight, value]
  );

  return (
    <textarea
      ref={ref}
      value={value}
      onChange={function (event) {
        onChange(event.target.value);
      }}
      rows={1}
      placeholder={placeholder}
      style={{
        width: "100%",
        resize: "none",
        minHeight,
        padding: theme.spacing.sm,
        border: `1px solid ${theme.colors.border}`,
        borderRadius: theme.radius.sm,
        fontSize: theme.fontSize.sm,
        background: theme.colors.background,
        overflow: "hidden",
        boxSizing: "border-box",
      }}
    />
  );
}
