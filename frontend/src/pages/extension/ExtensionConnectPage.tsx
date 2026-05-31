import { useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { createWebSession } from "../../api/auth";
import { getApiErrorMessage, getApiErrorStatus } from "../../api/errors";
import { approveExtensionPairing } from "../../api/extension";
import { Button } from "../../components/common/Button";
import { Input } from "../../components/common/Input";
import { AppShell } from "../../components/layout/AppShell";
import { theme } from "../../theme";

type ConnectStatus = "idle" | "success";

const userCodePattern = /^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{5}-?[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{5}$/;

function normalizeUserCode(value: string) {
  const compact = value.trim().toUpperCase().replace(/\s/g, "").replace("-", "");
  if (compact.length <= 5) {
    return compact;
  }
  return `${compact.slice(0, 5)}-${compact.slice(5, 10)}`;
}

function validateUserCode(value: string) {
  return userCodePattern.test(value.trim().toUpperCase().replace(/\s/g, ""));
}

function extensionConnectErrorMessage(error: unknown) {
  const status = getApiErrorStatus(error);
  const apiMessage = getApiErrorMessage(error).toLowerCase();

  if (status === 400) return "接続コードの形式が正しくありません";
  if (status === 401) return "ログインが必要です。もう一度ログインしてください";
  if (status === 403) return "この操作は許可されていません";
  if (status === 404) {
    if (apiMessage.includes("expired")) {
      return "接続コードの有効期限が切れました";
    }
    return "接続コードが見つかりません";
  }
  if (status === 409) return "すでに使用済みです";
  if (status === 410) return "接続コードの有効期限が切れました";
  if (status === 429) return "試行回数が多すぎます。しばらく待ってください";
  if (status >= 500) return "サーバー側の一時的な問題です。時間を置いてもう一度お試しください";
  if (status === 0) return "ネットワークに接続できませんでした";
  return "接続を承認できませんでした";
}

export function ExtensionConnectPage() {
  const [searchParams] = useSearchParams();
  const initialCode = useMemo(function () {
    return normalizeUserCode(searchParams.get("user_code") ?? searchParams.get("code") ?? "");
  }, [searchParams]);

  const [userCode, setUserCode] = useState(initialCode);
  const [fieldError, setFieldError] = useState("");
  const [message, setMessage] = useState("");
  const [status, setStatus] = useState<ConnectStatus>("idle");
  const [loading, setLoading] = useState(false);

  useEffect(
    function () {
      setUserCode(initialCode);
      setFieldError("");
      setMessage("");
      setStatus("idle");
    },
    [initialCode]
  );

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    const normalized = normalizeUserCode(userCode);
    setUserCode(normalized);
    setFieldError("");
    setMessage("");
    setStatus("idle");

    if (!validateUserCode(normalized)) {
      setFieldError("ABCDE-FGHJK の形式で入力してください");
      return;
    }

    setLoading(true);
    try {
      await createWebSession(true);
      await approveExtensionPairing(normalized);
      setStatus("success");
      setMessage("接続を承認しました。拡張機能に戻ってください。");
    } catch (error) {
      setMessage(extensionConnectErrorMessage(error));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AppShell>
      <div style={{ padding: theme.spacing.md, display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
        <section
          style={{
            border: `1px solid ${theme.colors.border}`,
            borderRadius: theme.radius.md,
            background: theme.colors.background,
            padding: theme.spacing.md,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.sm,
          }}
        >
          <div>
            <h1 style={{ margin: 0, fontSize: theme.fontSize.xl, fontWeight: 700 }}>Chrome拡張機能を接続</h1>
            <p style={{ margin: `${theme.spacing.xs} 0 0`, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
              拡張機能に表示された接続コードを確認して、このアカウントとの接続を承認します。
            </p>
          </div>

          <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
            <Input
              label="接続コード"
              value={userCode}
              onChange={function (value) {
                setUserCode(normalizeUserCode(value));
                setFieldError("");
              }}
              placeholder="ABCDE-FGHJK"
              error={fieldError}
              disabled={loading || status === "success"}
            />
            {message ? (
              <p
                role="status"
                style={{
                  margin: 0,
                  color: status === "success" ? theme.colors.success : theme.colors.danger,
                  fontSize: theme.fontSize.sm,
                }}
              >
                {message}
              </p>
            ) : null}
            <Button type="submit" loading={loading} disabled={status === "success"} fullWidth>
              接続を承認
            </Button>
          </form>
        </section>

        <section
          style={{
            border: `1px solid ${theme.colors.border}`,
            borderRadius: theme.radius.md,
            background: theme.colors.backgroundAlt,
            padding: theme.spacing.md,
            display: "flex",
            flexDirection: "column",
            gap: theme.spacing.xs,
          }}
        >
          <p style={{ margin: 0, fontWeight: 700 }}>接続後の流れ</p>
          <p style={{ margin: 0, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            Kindle Notebookで拡張機能からハイライトを取り込むと、問題タブで生成・復習できます。
          </p>
          <Link to="/?tab=question" style={{ color: theme.colors.primary, fontWeight: 700, fontSize: theme.fontSize.sm }}>
            問題タブへ
          </Link>
        </section>
      </div>
    </AppShell>
  );
}
