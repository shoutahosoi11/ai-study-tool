import { useState } from "react";
import { deleteAccount } from "../../api/users";
import {
  getReauthMethod,
  reauthenticateForSensitiveAction,
} from "../../api/auth";
import { getApiErrorStatus } from "../../api/errors";
import { Button } from "../../components/common/Button";
import { theme } from "../../theme";

const CONFIRM_PHRASE = "削除";

type DeleteAccountModalProps = {
  onClose: () => void;
  onDeleted: () => void;
};

export function DeleteAccountModal({ onClose, onDeleted }: DeleteAccountModalProps) {
  const reauthMethod = getReauthMethod();
  const [password, setPassword] = useState("");
  const [confirmText, setConfirmText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const confirmed = confirmText.trim() === CONFIRM_PHRASE;
  const needsPassword = reauthMethod === "password";
  const canSubmit = confirmed && (!needsPassword || password.length > 0) && !busy;

  async function handleDelete() {
    setError("");
    setBusy(true);
    try {
      await reauthenticateForSensitiveAction(needsPassword ? password : undefined);
    } catch (err) {
      setBusy(false);
      const code = (err as { code?: string }).code ?? "";
      if (code === "auth/wrong-password" || code === "auth/invalid-credential") {
        setError("パスワードが正しくありません");
      } else if (code === "auth/popup-closed-by-user" || code === "auth/cancelled-popup-request") {
        setError("再認証がキャンセルされました");
      } else if (code === "auth/too-many-requests") {
        setError("試行回数が多すぎます。しばらく待ってからお試しください");
      } else {
        setError("再認証に失敗しました");
      }
      return;
    }

    try {
      await deleteAccount();
      onDeleted();
    } catch (err) {
      const status = getApiErrorStatus(err);
      if (status === 403) {
        setError("このアカウントは管理者権限を持っているため削除できません");
      } else if (status === 401) {
        setError("再認証の有効期限が切れました。もう一度お試しください");
      } else {
        setError("アカウントの削除に失敗しました。時間をおいてお試しください");
      }
      setBusy(false);
    }
  }

  const inputStyle = {
    padding: theme.spacing.sm,
    borderRadius: theme.radius.sm,
    border: `1px solid ${theme.colors.border}`,
    fontSize: theme.fontSize.sm,
    background: theme.colors.background,
    color: "inherit",
    width: "100%",
    boxSizing: "border-box",
  } as const;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-account-title"
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0, 0, 0, 0.6)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: theme.spacing.md,
        zIndex: 50,
      }}
      onClick={function (event) {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
    >
      <div
        style={{
          width: "100%",
          maxWidth: "26rem",
          display: "flex",
          flexDirection: "column",
          gap: theme.spacing.md,
          padding: theme.spacing.lg,
          borderRadius: theme.radius.md,
          border: `1px solid ${theme.colors.danger}`,
          background: theme.colors.backgroundAlt,
        }}
      >
        <h3
          id="delete-account-title"
          style={{ margin: 0, fontSize: theme.fontSize.lg, color: theme.colors.danger }}
        >
          アカウントを削除しますか？
        </h3>
        <p style={{ margin: 0, fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
          この操作は取り消せません。次のデータがすべて完全に削除されます。
        </p>
        <ul
          style={{
            margin: 0,
            paddingLeft: theme.spacing.lg,
            fontSize: theme.fontSize.sm,
            color: theme.colors.secondary,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.xs,
          }}
        >
          <li>取り込んだハイライトと本の一覧</li>
          <li>生成された問題と解答履歴</li>
          <li>投稿・コメント・フォロー関係</li>
          <li>プロフィールと購読情報</li>
        </ul>

        {needsPassword && (
          <label style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
            <span style={{ fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
              確認のためパスワードを入力
            </span>
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={function (event) {
                setPassword(event.target.value);
              }}
              disabled={busy}
              style={inputStyle}
            />
          </label>
        )}
        {reauthMethod === "google" && (
          <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
            削除ボタンを押すと、確認のためGoogleの再ログイン画面が開きます。
          </p>
        )}

        <label style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
          <span style={{ fontSize: theme.fontSize.xs, color: theme.colors.secondary }}>
            「{CONFIRM_PHRASE}」と入力してください
          </span>
          <input
            type="text"
            value={confirmText}
            onChange={function (event) {
              setConfirmText(event.target.value);
            }}
            disabled={busy}
            placeholder={CONFIRM_PHRASE}
            style={inputStyle}
          />
        </label>

        {error && (
          <p style={{ margin: 0, fontSize: theme.fontSize.xs, color: theme.colors.danger }}>
            {error}
          </p>
        )}

        <div style={{ display: "flex", gap: theme.spacing.sm, justifyContent: "flex-end" }}>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            キャンセル
          </Button>
          <Button variant="danger" onClick={handleDelete} disabled={!canSubmit} loading={busy}>
            アカウントを完全に削除
          </Button>
        </div>
      </div>
    </div>
  );
}
