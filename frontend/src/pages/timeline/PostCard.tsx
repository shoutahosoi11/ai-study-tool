import { useState } from "react";
import { createPostComment, fetchPostQuestions, listPostComments } from "../../api/posts";
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

function formatCommentDate(dateStr: string): string {
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

function getPostLabel(post: TimelinePost): string {
  if (post.type === "question") {
    return "問題セットを共有";
  }
  return "投稿";
}

export function PostCard({ post }: Props) {
  const displayName = post.display_name || post.username;
  const [questionError, setQuestionError] = useState("");
  const [quizLoading, setQuizLoading] = useState(false);
  const [quizQuestions, setQuizQuestions] = useState<Question[]>([]);
  const [creatorExplanationNotes, setCreatorExplanationNotes] = useState<Record<string, string>>({});
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [commentsLoaded, setCommentsLoaded] = useState(false);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentsError, setCommentsError] = useState("");
  const [comments, setComments] = useState<PostComment[]>([]);
  const [commentDraft, setCommentDraft] = useState("");
  const [commentSubmitting, setCommentSubmitting] = useState(false);
  const [commentCount, setCommentCount] = useState(post.comment_count);

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

  function toggleComments() {
    const nextOpen = !commentsOpen;
    setCommentsOpen(nextOpen);
    if (nextOpen && !commentsLoaded) {
      void loadComments();
    }
  }

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
      setCommentsLoaded(true);
      setCommentsOpen(true);
      setCommentDraft("");
    } catch {
      setCommentsError("コメントの投稿に失敗しました");
    } finally {
      setCommentSubmitting(false);
    }
  }

  const hasQuestionCard = post.type === "question" && post.question_count > 0;

  return (
    <>
      <Card>
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

            {post.type !== 'question' && (
              <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
                {getPostLabel(post)}
              </p>
            )}

            {post.body && (
              <p style={{ margin: 0, fontSize: theme.fontSize.sm, whiteSpace: "pre-wrap" }}>
                {post.body}
              </p>
            )}

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
                <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
                  <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
                    問題セット
                  </p>
                  <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>
                    {post.book_title || "本の題名なし"}
                  </p>
                  <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
                    {post.question_count}問
                  </p>
                </div>
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

            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: theme.spacing.sm }}>
              <button
                type="button"
                onClick={toggleComments}
                style={{
                  border: "none",
                  background: "transparent",
                  padding: 0,
                  color: theme.colors.secondary,
                  fontSize: theme.fontSize.sm,
                  cursor: "pointer",
                }}
              >
                コメント {commentCount}
              </button>
            </div>

            {commentsOpen && (
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: theme.spacing.sm,
                  borderTop: `1px solid ${theme.colors.border}`,
                  paddingTop: theme.spacing.sm,
                }}
              >
                <textarea
                  value={commentDraft}
                  onChange={function (event) {
                    setCommentDraft(event.target.value);
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
                  <Button onClick={function () { void handleCreateComment(); }} loading={commentSubmitting}>
                    コメントする
                  </Button>
                </div>

                {commentsError && (
                  <p style={{ margin: 0, color: theme.colors.danger, fontSize: theme.fontSize.sm }}>
                    {commentsError}
                  </p>
                )}

                {commentsLoading ? (
                  <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
                    コメントを読み込み中...
                  </p>
                ) : comments.length === 0 ? (
                  <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
                    まだコメントがありません
                  </p>
                ) : (
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
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: theme.spacing.sm }}>
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
                          <p style={{ margin: 0, fontSize: theme.fontSize.sm, whiteSpace: "pre-wrap" }}>
                            {comment.content}
                          </p>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </Card>

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
