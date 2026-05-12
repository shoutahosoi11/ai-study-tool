import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { signUpWithEmail, auth } from "../../api/auth";
import { signUpBackendUser } from "../../api/users";
import { Button } from "../../components/common/Button";
import { Input } from "../../components/common/Input";
import { theme } from "../../theme";
import { getApiErrorMessage } from "../../api/errors";

export function SignupPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      if (!auth.currentUser) {
        await signUpWithEmail(email.trim(), password);
      }
      await signUpBackendUser(username);
      navigate("/");
    } catch (err: unknown) {
      const code = (err as { code?: string }).code ?? "";
      if (code === "auth/email-already-in-use") {
        setError("このメールアドレスは既に使用されています");
      } else if (code === "auth/weak-password") {
        setError("パスワードは6文字以上で入力してください");
      } else {
        const apiMessage = getApiErrorMessage(err);
        if (apiMessage === "user already exists") {
          setError("このユーザー名は既に使用されています");
        } else if (apiMessage) {
          setError(apiMessage);
        } else {
          setError("アカウント作成に失敗しました");
        }
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      style={{
        maxWidth: "480px",
        margin: "0 auto",
        padding: theme.spacing.xl,
        display: "flex",
        flexDirection: "column",
        gap: theme.spacing.lg,
        minHeight: "100vh",
        justifyContent: "center",
      }}
    >
      <h1 style={{ fontSize: theme.fontSize.xl, fontWeight: 700, margin: 0 }}>
        アカウント作成
      </h1>
      <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
        <Input label="ユーザー名" value={username} onChange={setUsername} placeholder="@username" />
        <Input label="メールアドレス" type="email" value={email} onChange={setEmail} />
        <Input label="パスワード" type="password" value={password} onChange={setPassword} />
        {error && (
          <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm, margin: 0 }}>
            {error}
          </p>
        )}
        <Button type="submit" loading={loading} fullWidth>
          アカウントを作成
        </Button>
      </form>
      <p style={{ textAlign: "center", fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
        ログインは{" "}
        <Link to="/login" style={{ color: theme.colors.primary, fontWeight: 600 }}>
          こちら
        </Link>
      </p>
    </div>
  );
}
