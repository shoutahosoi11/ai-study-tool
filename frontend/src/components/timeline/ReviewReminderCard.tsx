type Props = {
  onOpenQuestions: () => void;
};

export function ReviewReminderCard({ onOpenQuestions }: Props) {
  return (
    <section className="review-reminder-card" aria-label="復習リマインダー">
      <button type="button" className="review-reminder-card__button" onClick={onOpenQuestions} aria-label="問題タブを開く">
        <span className="review-reminder-card__mark" aria-hidden="true">
          ✓
        </span>
        <span className="review-reminder-card__copy">
          <strong>Daily review</strong>
          <span>saved / missed / book quiz</span>
        </span>
        <span className="review-reminder-card__arrow" aria-hidden="true">
          →
        </span>
      </button>
    </section>
  );
}
