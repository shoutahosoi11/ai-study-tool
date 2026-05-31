import type { ReactNode } from "react";

type Props = {
  title: string;
  count: number;
  onOpen: () => void;
  action?: ReactNode;
};

export function QuestionCard({ title, count, onOpen, action }: Props) {
  return (
    <div className="study-question-card">
      <button type="button" className="study-question-card__body" onClick={onOpen} aria-label={`${title} の問題を見る`}>
        <span className="study-question-card__icon" aria-hidden="true">
          ?
        </span>
        <span className="study-question-card__content">
          <strong>{title}</strong>
          <span>{count} Q</span>
        </span>
      </button>
      {action ? <div className="study-question-card__action">{action}</div> : null}
    </div>
  );
}
