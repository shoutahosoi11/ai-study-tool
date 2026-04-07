import { useState } from "react";
import { submitAnswer } from "../../api/questions";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { AnswerResult, Question } from "../../types/question";

type Phase = "input" | "submitting" | "result";

type Props = {
  question: Question;
  onClose: () => void;
};

export function AnswerModal({ question, onClose }: Props) {
  const [phase, setPhase] = useState<Phase>("input");
  const [selected, setSelected] = useState("");
  const [textAnswer, setTextAnswer] = useState("");
  const [result, setResult] = useState<AnswerResult | null>(null);
  const [error, setError] = useState("");

  async function handleSubmit() {
    const answer = question.question_type === "multiple_choice" ? selected : textAnswer;
    if (!answer) {
      return;
    }
    setPhase("submitting");
    setError("");
    try {
      const res = await submitAnswer(question.id, answer);
      setResult(res);
      setPhase("result");
    } catch {
      setError("送信に失敗しました");
      setPhase("input");
    }
  }

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.5)",
        display: "flex",
        alignItems: "flex-end",
        zIndex: 200,
      }}
      onClick={function (e) {
        if (e.target === e.currentTarget && phase !== "submitting") {
          onClose();
        }
      }}
    >
      <div
        style={{
          background: theme.colors.background,
          borderRadius: `${theme.radius.md} ${theme.radius.md} 0 0`,
          padding: theme.spacing.lg,
          width: "100%",
          maxWidth: "480px",
          margin: "0 auto",
          maxHeight: "80vh",
          overflowY: "auto",
        }}
      >
        <p style={{ fontWeight: 700, fontSize: theme.fontSize.base, marginBottom: theme.spacing.md }}>
          {question.content}
        </p>

        {phase === "result" && result ? (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            <p
              style={{
                fontWeight: 700,
                color: result.is_correct ? theme.colors.success : theme.colors.danger,
                fontSize: theme.fontSize.lg,
              }}
            >
              {result.is_correct ? "✓ 正解！" : "✗ 不正解"}
            </p>
            {!result.is_correct && (
              <p style={{ fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
                正解: {result.correct_answer}
              </p>
            )}
            {result.feedback && (
              <p style={{ fontSize: theme.fontSize.sm }}>{result.feedback}</p>
            )}
            <p style={{ fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
              {result.explanation}
            </p>
            <Button onClick={onClose} fullWidth>
              閉じる
            </Button>
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            {question.question_type === "multiple_choice" ? (
              question.options.map(function (opt) {
                return (
                  <button
                    key={opt}
                    onClick={function () {
                      setSelected(opt);
                    }}
                    style={{
                      padding: theme.spacing.sm,
                      border: `1px solid ${selected === opt ? theme.colors.primary : theme.colors.border}`,
                      borderRadius: theme.radius.sm,
                      background: selected === opt ? theme.colors.primary : theme.colors.background,
                      color: selected === opt ? theme.colors.background : theme.colors.primary,
                      cursor: "pointer",
                      textAlign: "left",
                      fontSize: theme.fontSize.sm,
                    }}
                  >
                    {opt}
                  </button>
                );
              })
            ) : (
              <textarea
                value={textAnswer}
                onChange={function (e) {
                  setTextAnswer(e.target.value);
                }}
                rows={4}
                placeholder="回答を入力..."
                style={{
                  padding: theme.spacing.sm,
                  border: `1px solid ${theme.colors.border}`,
                  borderRadius: theme.radius.sm,
                  fontSize: theme.fontSize.sm,
                  resize: "vertical",
                  width: "100%",
                }}
              />
            )}
            {error && <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{error}</p>}
            <Button
              onClick={handleSubmit}
              loading={phase === "submitting"}
              disabled={question.question_type === "multiple_choice" ? !selected : !textAnswer}
              fullWidth
            >
              回答する
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
