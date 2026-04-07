import { Avatar } from "../../components/common/Avatar";
import { Card } from "../../components/common/Card";
import { theme } from "../../theme";
import type { TimelinePost } from "../../types/post";

type Props = { post: TimelinePost };

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString("ja-JP", { month: "short", day: "numeric" });
}

function getPostLabel(post: TimelinePost): string {
  if (post.type === "question") {
    return "📝 問題を共有";
  }
  if (post.type === "note") {
    return "📖 ノートを共有";
  }
  if (post.type === "highlight" && post.book_title) {
    return `✏️ 「${post.book_title}」のハイライト`;
  }
  return "投稿";
}

export function PostCard({ post }: Props) {
  return (
    <Card>
      <div style={{ display: "flex", gap: theme.spacing.sm }}>
        <Avatar name={post.display_name || post.username} src={post.avatar_url} size={40} />
        <div style={{ flex: 1 }}>
          <div style={{ display: "flex", gap: theme.spacing.sm, alignItems: "center" }}>
            <span style={{ fontWeight: 700, fontSize: theme.fontSize.sm }}>{post.display_name}</span>
            <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>@{post.username}</span>
            <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs, marginLeft: "auto" }}>
              {formatDate(post.created_at)}
            </span>
          </div>
          <p style={{ margin: `${theme.spacing.xs} 0`, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
            {getPostLabel(post)}
          </p>
          <div style={{ display: "flex", gap: theme.spacing.lg, marginTop: theme.spacing.sm, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
            <span>💬 {post.comment_count}</span>
            <span>🔁 {post.repost_count}</span>
            <span>❤️ {post.like_count}</span>
          </div>
        </div>
      </div>
    </Card>
  );
}
