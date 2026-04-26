import { useEffect, useState } from "react";
import {
  createPostComment,
  fetchPostQuestions,
  likePost,
  listPostComments,
  repostPost,
  unlikePost,
  unrepostPost,
} from "../../api/posts";
import { Avatar } from "../../components/common/Avatar";
import { Button } from "../../components/common/Button";
import { Card } from "../../components/common/Card";
import { theme } from "../../theme";
import type { Question } from "../../types/question";
import type { PostComment, TimelinePost } from "../../types/post";
import { QuestionQuizSessionModal } from "../question/QuestionQuizSessionModal";

type Props = { post: TimelinePost };

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString("ja-JP", { month: "short", day: "numeric" });
}

function formatCommentDate(dateStr: string) {
  const date = new Date(dateStr);
  return date.toLocaleString("ja-JP", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

function toQuestionShape(postQuestion: {
  id: string;
  question_type: "multiple_choice" | "descriptive";
  content: string;
  options: string[];
  correct_answer: string;
  explanation: string;
}): Question {
  return {
    id: postQuestion.id,
    question_type: postQuestion.question_type,
    content: postQuestion.content,
    options: postQuestion.options,
    correct_answer: postQuestion.correct_answer,
    explanation: postQuestion.explanation,
  };
}

export function PostCard({ post }: Props) {
  const displayName = post.display_name || post.username;
  const [questionError, setQuestionError] = useState("");
  const [quizLoading, setQuizLoading] = useState(false);
  const [quizQuestions, setQuizQuestions] = useState<Question[]>([]);
  const [creatorExplanationNotes, setCreatorExplanationNotes] = useState<Record<string, string>>({});
  const [detailOpen, setDetailOpen] = useState(false);
  const [commentsLoaded, setCommentsLoaded] = useState(false);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentsError, setCommentsError] = useState("");
  const [comments, setComments] = useState<PostComment[]>([]);
  const [commentDraft, setCommentDraft] = useState("");
  const [commentSubmitting, setCommentSubmitting] = useState(false);
  const [commentCount, setCommentCount] = useState(post.comment_count);
  const [likeCount, setLikeCount] = useState(post.like_count);
  const [repostCount, setRepostCount] = useState(post.repost_count);
  const [liked, setLiked] = useState(false);
  const [reposted, setReposted] = useState(false);
  const [likeBusy, setLikeBusy] = useState(false);
  const [repostBusy, setRepostBusy] = useState(false);
  const [actionError, setActionError] = useState("");

  const hasQuestionCard = post.type === "question" && post.question_count > 0;

  async function handleSolveQuestions() {
    setQuestionError("");
    setQuizLoading(true);
    setQuizQuestions([]);
    setCreatorExplanationNotes({});
    try {
      const postedQuestions = await fetchPostQuestions(post.id);
      if (postedQuestions.length === 0) {
        setQuestionError("この投稿の問題はまだ取得できません");
        return;
      }
      setQuizQuestions(postedQuestions.map(toQuestionShape));
      setCreatorExplanationNotes(
        postedQuestions.reduce(function (acc, question) {
          acc[question.id] = question.note ?? "";
          return acc;
        }, {} as Record<string, string>)
      );
    } catch {
      setQuestionError("投稿された問題の取得に失敗しました");
    } finally {
      setQuizLoading(false);
    }
  }

  async function loadComments() {
    setCommentsLoading(true);
    setCommentsError("");
    try {
      const nextComments = await listPostComments(post.id);
      setComments(nextComments);
      setCommentsLoaded(true);
    } catch {
      setCommentsError("コメントの取得に失敗しました");
    } finally {
      setCommentsLoading(false);
    }
  }

  useEffect(
    function () {
      if (!detailOpen || commentsLoaded) {
        return;
      }
      void loadComments();
    },
    [commentsLoaded, detailOpen]
  );

  async function handleCreateComment() {
    const trimmed = commentDraft.trim();
    if (!trimmed) {
      return;
    }

    setCommentSubmitting(true);
    setCommentsError("");
    try {
      const created = await createPostComment(post.id, trimmed);
      setComments(function (prev) {
        return [created, ...prev];
      });
      setCommentCount(function (prev) {
        return prev + 1;
      });
      setCommentDraft("");
      setCommentsLoaded(true);
      setDetailOpen(true);
    } catch {
      setCommentsError("コメントの投稿に失敗しました");
    } finally {
      setCommentSubmitting(false);
    }
  }

  async function handleToggleLike() {
    setActionError("");
    setLikeBusy(true);
    try {
      if (liked) {
        await unlikePost(post.id);
        setLiked(false);
        setLikeCount(function (prev) {
          return Math.max(prev - 1, 0);
        });
      } else {
        await likePost(post.id);
        setLiked(true);
        setLikeCount(function (prev) {
          return prev + 1;
        });
      }
    } catch {
      setActionError("いいねの更新に失敗しました");
    } finally {
      setLikeBusy(false);
    }
  }

  async function handleToggleRepost() {
    setActionError("");
    setRepostBusy(true);
    try {
      if (reposted) {
        await unrepostPost(post.id);
        setReposted(false);
        setRepostCount(function (prev) {
          return Math.max(prev - 1, 0);
        });
      } else {
        await repostPost(post.id);
        setReposted(true);
        setRepostCount(function (prev) {
          return prev + 1;
        });
      }
    } catch {
      setActionError("リポストの更新に失敗しました");
    } finally {
      setRepostBusy(false);
    }
  }

  function openDetail() {
    setDetailOpen(true);
  }

  return (
    <>
      <Card>
        <div style={{ display: "flex", gap: theme.spacing.sm }}>
          <Avatar name={displayName} src={post.avatar_url} size={40} />
          <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
            <button
              type="button"
              onClick={openDetail}
              style={{
                border: "none",
                background: "transparent",
                padding: 0,
                textAlign: "left",
                cursor: "pointer",
                display: "flex",
                flexDirection: "column",
                gap: theme.spacing.sm,
              }}
            >
              <div style={{ display: "flex", gap: theme.spacing.sm, alignItems: "center" }}>
                <span style={{ fontWeight: 700, fontSize: theme.fontSize.sm }}>{displayName}</span>
                <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>@{post.username}</span>
                <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs, marginLeft: "auto" }}>
                  {formatDate(post.created_at)}
                </span>
              </div>

              {post.body && (
                <p style={{ margin: 0, fontSize: theme.fontSize.sm, whiteSpace: "pre-wrap", color: "#0f1419" }}>
                  {post.body}
                </p>
              )}
            </button>

            {hasQuestionCard && (
              <div
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
                <button
                  type="button"
                  onClick={openDetail}
                  style={{
                    border: "none",
                    background: "transparent",
                    padding: 0,
                    textAlign: "left",
                    cursor: "pointer",
                    display: "flex",
                    flexDirection: "column",
                    gap: theme.spacing.xs,
                  }}
                >
                  <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                    問題セット
                  </p>
                  <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
                    {post.book_title || "本の題名なし"}
                  </p>
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
                    {post.question_count}問
                  </p>
                  <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                    クリックでコメントと詳細を見る
                  </p>
                </button>
                <div style={{ display: "flex", justifyContent: "flex-end" }}>
                  <Button onClick={function () { void handleSolveQuestions(); }} loading={quizLoading}>
                    この問題を解く
                  </Button>
                </div>
              </div>
            )}

            {questionError && (
              <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
                {questionError}
              </p>
            )}
            {actionError && (
              <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
                {actionError}
              </p>
            )}

            <div style={{ display: "flex", gap: theme.spacing.sm, flexWrap: "wrap" }}>
              <ActionButton label={repostBusy ? "処理中..." : `リポスト ${repostCount}`} active={reposted} onClick={handleToggleRepost} />
              <ActionButton label={likeBusy ? "処理中..." : `いいね ${likeCount}`} active={liked} onClick={handleToggleLike} />
              <ActionButton label={`コメント ${commentCount}`} active={false} onClick={openDetail} />
            </div>
          </div>
        </div>
      </Card>

      {detailOpen && (
        <PostDetailModal
          post={post}
          displayName={displayName}
          questionError={questionError}
          actionError={actionError}
          solving={quizLoading}
          onClose={function () {
            setDetailOpen(false);
          }}
          onSolve={function () {
            void handleSolveQuestions();
          }}
          liked={liked}
          reposted={reposted}
          likeBusy={likeBusy}
          repostBusy={repostBusy}
          likeCount={likeCount}
          repostCount={repostCount}
          commentCount={commentCount}
          onToggleLike={handleToggleLike}
          onToggleRepost={handleToggleRepost}
          commentsLoaded={commentsLoaded}
          commentsLoading={commentsLoading}
          commentsError={commentsError}
          comments={comments}
          commentDraft={commentDraft}
          onChangeCommentDraft={setCommentDraft}
          commentSubmitting={commentSubmitting}
          onSubmitComment={function () {
            void handleCreateComment();
          }}
        />
      )}

      {(quizLoading || quizQuestions.length > 0) && (
        <QuestionQuizSessionModal
          bookTitle={post.book_title || "共有された問題"}
          questions={quizQuestions}
          loading={quizLoading}
          readonlyExplanationNotes={creatorExplanationNotes}
          mergedNoteLabel="作成者の解説"
          onClose={function () {
            setQuizLoading(false);
            setQuizQuestions([]);
            setCreatorExplanationNotes({});
          }}
        />
      )}
    </>
  );
}

function ActionButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        border: `1px solid ${active ? theme.colors.primary : theme.colors.border}`,
        borderRadius: theme.radius.full,
        background: active ? "#e8f5fd" : theme.colors.background,
        color: active ? "#1d9bf0" : "#0f1419",
        padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
        fontSize: theme.fontSize.sm,
        fontWeight: 700,
        cursor: "pointer",
      }}
    >
      {label}
    </button>
  );
}

function PostDetailModal({
  post,
  displayName,
  questionError,
  actionError,
  solving,
  onClose,
  onSolve,
  liked,
  reposted,
  likeBusy,
  repostBusy,
  likeCount,
  repostCount,
  commentCount,
  onToggleLike,
  onToggleRepost,
  commentsLoaded,
  commentsLoading,
  commentsError,
  comments,
  commentDraft,
  onChangeCommentDraft,
  commentSubmitting,
  onSubmitComment,
}: {
  post: TimelinePost;
  displayName: string;
  questionError: string;
  actionError: string;
  solving: boolean;
  onClose: () => void;
  onSolve: () => void;
  liked: boolean;
  reposted: boolean;
  likeBusy: boolean;
  repostBusy: boolean;
  likeCount: number;
  repostCount: number;
  commentCount: number;
  onToggleLike: () => void;
  onToggleRepost: () => void;
  commentsLoaded: boolean;
  commentsLoading: boolean;
  commentsError: string;
  comments: PostComment[];
  commentDraft: string;
  onChangeCommentDraft: (value: string) => void;
  commentSubmitting: boolean;
  onSubmitComment: () => void;
}) {
  const hasQuestionCard = post.type === "question" && post.question_count > 0;

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(15, 20, 25, 0.45)",
        display: "flex",
        alignItems: "flex-end",
        justifyContent: "center",
        padding: theme.spacing.md,
        zIndex: 230,
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
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>投稿の詳細</p>
          </div>
          <Button variant="ghost" onClick={onClose}>
            閉じる
          </Button>
        </div>

        <div style={{ display: "flex", gap: theme.spacing.sm }}>
          <Avatar name={displayName} src={post.avatar_url} size={40} />
          <div style={{ flex: 1, display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
            <div style={{ display: "flex", gap: theme.spacing.sm, alignItems: "center" }}>
              <span style={{ fontWeight: 700, fontSize: theme.fontSize.sm }}>{displayName}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>@{post.username}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs, marginLeft: "auto" }}>
                {formatDate(post.created_at)}
              </span>
            </div>
            {post.body && (
              <p style={{ margin: 0, fontSize: theme.fontSize.sm, whiteSpace: "pre-wrap" }}>{post.body}</p>
            )}
          </div>
        </div>

        {hasQuestionCard && (
          <div
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
            <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
              問題セット
            </p>
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
              {post.book_title || "本の題名なし"}
            </p>
            <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
              {post.question_count}問
            </p>
            <div style={{ display: "flex", justifyContent: "flex-end" }}>
              <Button onClick={onSolve} loading={solving}>
                この問題を解く
              </Button>
            </div>
          </div>
        )}

        {questionError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{questionError}</p>}
        {actionError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{actionError}</p>}

        <div style={{ display: "flex", gap: theme.spacing.sm, flexWrap: "wrap" }}>
          <ActionButton label={repostBusy ? "処理中..." : `リポスト ${repostCount}`} active={reposted} onClick={onToggleRepost} />
          <ActionButton label={likeBusy ? "処理中..." : `いいね ${likeCount}`} active={liked} onClick={onToggleLike} />
          <ActionButton label={`コメント ${commentCount}`} active={false} onClick={function () {}} />
        </div>

        <div
          style={{
            borderTop: `1px solid ${theme.colors.border}`,
            paddingTop: theme.spacing.md,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.sm,
          }}
        >
          <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>コメント</p>
          <textarea
            value={commentDraft}
            onChange={function (event) {
              onChangeCommentDraft(event.target.value);
            }}
            rows={3}
            placeholder="コメントを書く"
            style={{
              width: "100%",
              resize: "vertical",
              padding: theme.spacing.sm,
              border: `1px solid ${theme.colors.border}`,
              borderRadius: theme.radius.sm,
              fontSize: theme.fontSize.sm,
              background: theme.colors.background,
            }}
          />
          <div style={{ display: "flex", justifyContent: "flex-end" }}>
            <Button onClick={onSubmitComment} loading={commentSubmitting} disabled={!commentDraft.trim()}>
              コメントする
            </Button>
          </div>

          {commentsError && <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>{commentsError}</p>}
          {commentsLoading && <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>コメントを読み込み中...</p>}
          {!commentsLoading && commentsLoaded && comments.length === 0 ? (
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>まだコメントがありません</p>
          ) : null}
          {!commentsLoading && comments.length > 0 ? (
            <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.sm }}>
              {comments.map(function (comment) {
                return (
                  <div
                    key={comment.id}
                    style={{
                      borderRadius: theme.radius.sm,
                      background: theme.colors.backgroundAlt,
                      border: `1px solid ${theme.colors.border}`,
                      padding: theme.spacing.sm,
                      display: "flex",
                      flexDirection: "column",
                      gap: theme.spacing.xs,
                    }}
                  >
                    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: theme.spacing.sm }}>
                      <div style={{ display: "flex", alignItems: "center", gap: theme.spacing.sm }}>
                        <Avatar name={comment.display_name || comment.username} src={comment.avatar_url} size={28} />
                        <div style={{ display: "flex", flexDirection: "column", gap: "2px" }}>
                          <span style={{ fontWeight: 700, fontSize: theme.fontSize.xs }}>
                            {comment.display_name || comment.username}
                          </span>
                          <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
                            @{comment.username}
                          </span>
                        </div>
                      </div>
                      <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
                        {formatCommentDate(comment.created_at)}
                      </span>
                    </div>
                    <p style={{ margin: 0, fontSize: theme.fontSize.sm, whiteSpace: "pre-wrap" }}>{comment.content}</p>
                  </div>
                );
              })}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
