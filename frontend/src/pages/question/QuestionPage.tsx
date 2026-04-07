import { useEffect, useState } from "react";
import { listQuestions } from "../../api/questions";
import { Spinner } from "../../components/common/Spinner";
import { theme } from "../../theme";
import type { Question } from "../../types/question";
import { AnswerModal } from "./AnswerModal";
import { QuestionCard } from "./QuestionCard";

export function QuestionPage() {
  const [questions, setQuestions] = useState<Question[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Question | null>(null);

  useEffect(function () {
    setLoading(true);
    listQuestions()
      .then(setQuestions)
      .catch(function () {
        setError("問題の取得に失敗しました");
      })
      .finally(function () {
        setLoading(false);
      });
  }, []);

  return (
    <div style={{ padding: theme.spacing.md }}>
      <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: `0 0 ${theme.spacing.md}` }}>
        問題
      </h2>
      {error && <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{error}</p>}
      {loading ? (
        <div style={{ display: "flex", justifyContent: "center", padding: theme.spacing.lg }}>
          <Spinner />
        </div>
      ) : questions.length === 0 ? (
        <p style={{ textAlign: "center", color: theme.colors.secondary, padding: theme.spacing.xl }}>
          まだ問題がありません
        </p>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
          {questions.map(function (q) {
            return (
              <QuestionCard
                key={q.id}
                question={q}
                onAnswer={function () {
                  setSelected(q);
                }}
              />
            );
          })}
        </div>
      )}
      {selected && (
        <AnswerModal
          question={selected}
          onClose={function () {
            setSelected(null);
          }}
        />
      )}
    </div>
  );
}
