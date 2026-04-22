import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { listIncorrectQuestions, listSavedQuestions } from "../../api/questions";
import { getMe, updateQuestionSettings } from "../../api/users";
import { signOutUser } from "../../api/auth";
import { Avatar } from "../../components/common/Avatar";
import { Button } from "../../components/common/Button";
import { Spinner } from "../../components/common/Spinner";
import { IncorrectQuestionsModal } from "./IncorrectQuestionsModal";
import { SavedQuestionsModal } from "./SavedQuestionsModal";
import { theme } from "../../theme";
import type { MeResponse } from "../../types/user";
import type { IncorrectQuestion, SavedQuestion } from "../../types/question";

type UserProfile = MeResponse & {
  follower_count?: number;
  following_count?: number;
};

function resolveDefaultQuestionCount(value?: number) {
  return typeof value === "number" ? value : 3;
}

export function ProfilePage() {
  const navigate = useNavigate();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [defaultQuestionCount, setDefaultQuestionCount] = useState(3);
  const [savingSettings, setSavingSettings] = useState(false);
  const [settingsMessage, setSettingsMessage] = useState("");
  const [savedQuestionsOpen, setSavedQuestionsOpen] = useState(false);
  const [savedQuestions, setSavedQuestions] = useState<SavedQuestion[]>([]);
  const [savedQuestionsLoading, setSavedQuestionsLoading] = useState(false);
  const [savedQuestionsError, setSavedQuestionsError] = useState("");
  const [incorrectQuestionsOpen, setIncorrectQuestionsOpen] = useState(false);
  const [incorrectQuestions, setIncorrectQuestions] = useState<IncorrectQuestion[]>([]);
  const [incorrectQuestionsLoading, setIncorrectQuestionsLoading] = useState(false);
  const [incorrectQuestionsError, setIncorrectQuestionsError] = useState("");

  useEffect(function () {
    getMe()
      .then(function (res) {
        setProfile(res);
        setDefaultQuestionCount(resolveDefaultQuestionCount(res.default_question_count));
      })
      .catch(function () {
        setError("プロフィールの取得に失敗しました");
      })
      .finally(function () {
        setLoading(false);
      });
  }, []);

  async function handleSignOut() {
    await signOutUser();
    navigate("/login");
  }

  async function handleSaveQuestionSettings() {
    setSettingsMessage("");
    setSavingSettings(true);
    try {
      const updated = await updateQuestionSettings(defaultQuestionCount);
      setProfile(updated);
      setDefaultQuestionCount(resolveDefaultQuestionCount(updated.default_question_count));
      setSettingsMessage("既定の出題数を保存しました");
    } catch {
      setSettingsMessage("既定の出題数の保存に失敗しました");
    } finally {
      setSavingSettings(false);
    }
  }

  async function handleOpenSavedQuestions() {
    setSavedQuestionsOpen(true);
    setSavedQuestionsLoading(true);
    setSavedQuestionsError("");
    try {
      const questions = await listSavedQuestions();
      setSavedQuestions(questions);
    } catch {
      setSavedQuestionsError("保存済み問題の取得に失敗しました");
    } finally {
      setSavedQuestionsLoading(false);
    }
  }

  async function handleOpenIncorrectQuestions() {
    setIncorrectQuestionsOpen(true);
    setIncorrectQuestionsLoading(true);
    setIncorrectQuestionsError("");
    try {
      const questions = await listIncorrectQuestions();
      setIncorrectQuestions(questions);
    } catch {
      setIncorrectQuestionsError("間違った問題の取得に失敗しました");
    } finally {
      setIncorrectQuestionsLoading(false);
    }
  }

  if (loading) {
    return (
      <div style={{ display: "flex", justifyContent: "center", padding: theme.spacing.xl }}>
        <Spinner />
      </div>
    );
  }

  if (error) {
    return <p style={{ color: theme.colors.danger, padding: theme.spacing.md }}>{error}</p>;
  }

  return (
    <div style={{ padding: theme.spacing.md }}>
      <h2 style={{ fontSize: theme.fontSize.lg, fontWeight: 700, margin: `0 0 ${theme.spacing.lg}` }}>
        プロフィール
      </h2>
      {profile && (
        <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.lg }}>
          <div style={{ display: "flex", alignItems: "center", gap: theme.spacing.md }}>
            <Avatar name={profile.display_name || profile.username} src={profile.avatar_url} size={64} />
            <div>
              <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.lg }}>{profile.display_name}</p>
              <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>@{profile.username}</p>
            </div>
          </div>
          {profile.bio && (
            <p style={{ margin: 0, fontSize: theme.fontSize.sm }}>{profile.bio}</p>
          )}
          <div style={{ display: "flex", gap: theme.spacing.xl }}>
            <div>
              <span style={{ fontWeight: 700 }}>{profile.following_count ?? 0}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.sm, marginLeft: theme.spacing.xs }}>フォロー</span>
            </div>
            <div>
              <span style={{ fontWeight: 700 }}>{profile.follower_count ?? 0}</span>
              <span style={{ color: theme.colors.secondary, fontSize: theme.fontSize.sm, marginLeft: theme.spacing.xs }}>フォロワー</span>
            </div>
          </div>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.sm,
              padding: theme.spacing.md,
              borderRadius: theme.radius.md,
              border: `1px solid ${theme.colors.border}`,
              background: theme.colors.background,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.sm }}>既定の出題数</p>
            <select
              value={defaultQuestionCount}
              onChange={function (event) {
                setDefaultQuestionCount(Number(event.target.value));
              }}
              style={{
                padding: theme.spacing.sm,
                borderRadius: theme.radius.sm,
                border: `1px solid ${theme.colors.border}`,
                fontSize: theme.fontSize.sm,
                background: theme.colors.background,
              }}
            >
              {[1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map(function (count) {
                return (
                  <option key={count} value={count}>
                    {count}問
                  </option>
                );
              })}
              <option value={0}>全部（安全のため最大20問）</option>
            </select>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
              「問題を作る」を押した時は、この設定がそのまま使われます。
            </p>
            {settingsMessage && (
              <p
                style={{
                  margin: 0,
                  color: settingsMessage.indexOf("失敗") >= 0 ? theme.colors.danger : theme.colors.success,
                  fontSize: theme.fontSize.xs,
                }}
              >
                {settingsMessage}
              </p>
            )}
            <div style={{ display: "flex", justifyContent: "flex-end" }}>
              <Button onClick={handleSaveQuestionSettings} loading={savingSettings}>
                保存
              </Button>
            </div>
          </div>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.sm,
              padding: theme.spacing.md,
              borderRadius: theme.radius.md,
              border: `1px solid ${theme.colors.border}`,
              background: theme.colors.background,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.sm }}>保存済み問題</p>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
              解き終わったあとに保存した問題と、自分のメモを見返せます。
            </p>
            <div style={{ display: "flex", justifyContent: "flex-end" }}>
              <Button variant="outline" onClick={handleOpenSavedQuestions}>
                保存済み問題を見る
              </Button>
            </div>
          </div>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: theme.spacing.sm,
              padding: theme.spacing.md,
              borderRadius: theme.radius.md,
              border: `1px solid ${theme.colors.border}`,
              background: theme.colors.background,
            }}
          >
            <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.sm }}>間違った問題</p>
            <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.xs }}>
              直近で不正解だった問題を一覧で見返せます。正解し直すと、この一覧から消えます。
            </p>
            <div style={{ display: "flex", justifyContent: "flex-end" }}>
              <Button variant="outline" onClick={handleOpenIncorrectQuestions}>
                間違った問題を見る
              </Button>
            </div>
          </div>
          <Button variant="outline" onClick={handleSignOut}>
            ログアウト
          </Button>
        </div>
      )}
      {savedQuestionsOpen && (
        <SavedQuestionsModal
          questions={savedQuestions}
          loading={savedQuestionsLoading}
          error={savedQuestionsError}
          onClose={function () {
            setSavedQuestionsOpen(false);
          }}
        />
      )}
      {incorrectQuestionsOpen && (
        <IncorrectQuestionsModal
          questions={incorrectQuestions}
          loading={incorrectQuestionsLoading}
          error={incorrectQuestionsError}
          onClose={function () {
            setIncorrectQuestionsOpen(false);
          }}
        />
      )}
    </div>
  );
}
