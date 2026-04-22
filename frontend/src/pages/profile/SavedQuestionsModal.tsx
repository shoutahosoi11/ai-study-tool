import { Button } from "../../components/common/Button";
import { theme } from "../../theme";
import type { SavedQuestion } from "../../types/question";

type Props = {
  questions: SavedQuestion[];
  loading: boolean;
  error: string;
  onClose: () => void;
};

function formatSavedAt(savedAt: string) {
  const date = new Date(savedAt);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return date.toLocaleString("ja-JP");
}

export function SavedQuestionsModal({ questions, loading, error, onClose }: Props) {
  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.45)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: theme.spacing.md,
        zIndex: 240,
      }}
      onClick={function (event) {
        if (event.target === event.currentTarget) {
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
              プロフィール
            </p>
            <p style={{ margin: 0, fontSize: theme.fontSize.base, fontWeight: 700 }}>
              保存済み問題
            </p>
          </div>
          <Button variant="ghost" onClick={onClose}>
            閉じる
          </Button>
        </div>

        {loading ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            読み込み中...
          </p>
        ) : error ? (
          <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
            {error}
          </p>
        ) : questions.length === 0 ? (
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            まだ保存した問題がありません
          </p>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            {questions.map(function (question, index) {
              return (
                <div
                  key={question.id}
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
                  <div style={{ display: "flex", justifyContent: "space-between", gap: theme.spacing.md, alignItems: "center" }}>
                    <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
                      保存済み問題 {index + 1}
                    </p>
                    <span
                      style={{
                        fontSize: theme.fontSize.xs,
                        color: theme.colors.secondary,
                        border: `1px solid ${theme.colors.border}`,
                        borderRadius: theme.radius.full,
                        padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
                        whiteSpace: "nowrap",
                      }}
                    >
                      {question.question_type === "multiple_choice" ? "選択式" : "記述式"}
                    </span>
                  </div>
                  <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
                    {question.content}
                  </p>
                  {question.options.length > 0 && (
                    <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
                      {question.options.map(function (option) {
                        return (
                          <p key={option} style={{ margin: 0, fontSize: theme.fontSize.sm }}>
                            ・{option}
                          </p>
                        );
                      })}
                    </div>
                  )}
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>
                    正解: {question.correct_answer}
                  </p>
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
                    解説: {question.explanation}
                  </p>
                  {question.note && (
                    <div
                      style={{
                        borderRadius: theme.radius.sm,
                        background: theme.colors.background,
                        border: `1px solid ${theme.colors.border}`,
                        padding: theme.spacing.sm,
                      }}
                    >
                      <p style={{ margin: `0 0 ${theme.spacing.xs}`, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                        自分のメモ
                      </p>
                      <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>{question.note}</p>
                    </div>
                  )}
                  <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                    保存日時: {formatSavedAt(question.saved_at)}
                  </p>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
