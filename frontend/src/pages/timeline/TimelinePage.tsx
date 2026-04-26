import { Button } from "../../components/common/Button";
import { Spinner } from "../../components/common/Spinner";
import { useTimeline } from "../../hooks/useTimeline";
import { theme } from "../../theme";
import { PostCard } from "./PostCard";

export function TimelinePage() {
  const { posts, loading, error, loadMore, hasMore } = useTimeline();

  return (
    <div style={{ padding: theme.spacing.md }}>
      <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: `0 0 ${theme.spacing.md}` }}>
        タイムライン
      </h2>
      {error && (
        <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{error}</p>
      )}
      <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
        {posts.map(function (post) {
          return <PostCard key={post.id} post={post} />;
        })}
      </div>
      {loading && (
        <div style={{ display: "flex", justifyContent: "center", padding: theme.spacing.lg }}>
          <Spinner />
        </div>
      )}
      {!loading && posts.length === 0 && (
        <p style={{ textAlign: "center", color: theme.colors.secondary, padding: theme.spacing.xl }}>
          まだ投稿がありません
        </p>
      )}
      {!loading && hasMore && (
        <div style={{ display: "flex", justifyContent: "center", marginTop: theme.spacing.md }}>
          <Button variant="outline" onClick={loadMore}>
            もっと見る
          </Button>
        </div>
      )}
    </div>
  );
}
