import { useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { signInWithEmail } from "../../api/auth";
import { Button } from "../../components/common/Button";
import { Input } from "../../components/common/Input";
import { theme } from "../../theme";

export function LoginPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await signInWithEmail(email, password);
      navigate("/");
    } catch (err: unknown) {
      const code = (err as { code?: string }).code ?? "";
      if (code === "auth/wrong-password" || code === "auth/user-not-found") {
        setError("メールアドレスまたはパスワードが間違っています");
      } else if (code === "auth/email-not-verified") {
        setError("メールアドレスが確認されていません");
      } else {
        setError("ログインに失敗しました");
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
        ログイン
      </h1>
      <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: theme.spacing.md }}>
        <Input label="メールアドレス" type="email" value={email} onChange={setEmail} />
        <Input label="パスワード" type="password" value={password} onChange={setPassword} />
        {error && (
          <p style={{ color: theme.colors.danger, fontSize: theme.fontSize.sm, margin: 0 }}>
            {error}
          </p>
        )}
        <Button type="submit" loading={loading} fullWidth>
          ログイン
        </Button>
      </form>
      <p style={{ textAlign: "center", fontSize: theme.fontSize.sm, color: theme.colors.secondary }}>
        アカウントをお持ちでない方は{" "}
        <Link to="/signup" style={{ color: theme.colors.primary, fontWeight: 600 }}>
          こちら
        </Link>
      </p>
    </div>
  );
}
