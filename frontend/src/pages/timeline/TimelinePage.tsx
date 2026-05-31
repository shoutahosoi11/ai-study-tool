import { useNavigate } from "react-router-dom";
import { Button } from "../../components/common/Button";
import { MainTimeline } from "../../components/layout/MainTimeline";
import { EmptyState } from "../../components/timeline/EmptyState";
import { LoadingState } from "../../components/timeline/LoadingState";
import { ReviewReminderCard } from "../../components/timeline/ReviewReminderCard";
import { StudyPostCard } from "../../components/timeline/StudyPostCard";
import { useTimeline } from "../../hooks/useTimeline";
import { theme } from "../../theme";

export function TimelinePage() {
  const { posts, loading, error, loadMore, hasMore } = useTimeline();
  const navigate = useNavigate();

  return (
    <MainTimeline
      title="Home"
      actions={
        <button
          type="button"
          className="main-timeline__icon-button"
          aria-label="問題を開く"
          onClick={function () {
            navigate("/?tab=question");
          }}
        >
          +
        </button>
      }
    >
      <div className="timeline-composer" aria-label="クイックアクション">
        <button
          type="button"
          className="timeline-composer__button"
          onClick={function () {
            navigate("/?tab=question");
          }}
          aria-label="問題を開く"
        >
          <span aria-hidden="true">?</span>
          <strong>Quiz</strong>
        </button>
        <button
          type="button"
          className="timeline-composer__button"
          onClick={function () {
            navigate("/?tab=profile");
          }}
          aria-label="プロフィールを開く"
        >
          <span aria-hidden="true">✓</span>
          <strong>Log</strong>
        </button>
      </div>
      <ReviewReminderCard
        onOpenQuestions={function () {
          navigate("/?tab=question");
        }}
      />
      {error && (
        <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm, padding: `0 ${theme.spacing.md}` }}>{error}</p>
      )}
      <div className="timeline-feed" aria-label="学習投稿">
        {posts.map(function (post) {
          return <StudyPostCard key={post.id} post={post} />;
        })}
      </div>
      {loading && <LoadingState />}
      {!loading && posts.length === 0 && (
        <EmptyState title="No posts" description="Questions and reviews will appear here." />
      )}
      {!loading && hasMore && (
        <div style={{ display: "flex", justifyContent: "center", marginTop: theme.spacing.md }}>
          <Button variant="outline" onClick={loadMore}>
            More
          </Button>
        </div>
      )}
    </MainTimeline>
  );
}
