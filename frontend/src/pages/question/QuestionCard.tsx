import { Card } from "../../components/common/Card";
import { theme } from "../../theme";
import type { Question } from "../../types/question";

type Props = {
  question: Question;
  onAnswer: () => void;
};

export function QuestionCard({ question, onAnswer }: Props) {
  const label = question.question_type === "multiple_choice" ? "選択" : "記述";

  return (
    <Card onClick={onAnswer}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <p style={{ margin: 0, fontSize: theme.fontSize.sm, flex: 1, paddingRight: theme.spacing.sm }}>
          {question.content}
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
          {label}
        </span>
      </div>
    </Card>
  );
}
