import { Link } from "react-router-dom";
import { theme } from "../../theme";

const steps = [
  "Kindle Notebookを開く",
  "Chrome拡張機能を接続する",
  "ハイライトを取り込む",
  "問題を生成する",
  "解いて復習する",
];

export function OnboardingGuide() {
  return (
    <section
      style={{
        border: `1px solid ${theme.colors.border}`,
        borderRadius: theme.radius.md,
        background: theme.colors.background,
        padding: theme.spacing.md,
        display: "flex",
        flexDirection: "column",
        gap: theme.spacing.sm,
        margin: theme.spacing.md,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", gap: theme.spacing.md, alignItems: "center" }}>
        <div>
          <p style={{ margin: 0, fontWeight: 700, fontSize: theme.fontSize.base }}>学習を始める</p>
          <p style={{ margin: `${theme.spacing.xs} 0 0`, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
            取り込みから復習まで、この順番で進めます。
          </p>
        </div>
        <Link
          to="/extension/connect"
          style={{
            color: theme.colors.primary,
            fontSize: theme.fontSize.sm,
            fontWeight: 700,
            whiteSpace: "nowrap",
          }}
        >
          接続
        </Link>
      </div>
      <ol style={{ margin: 0, paddingLeft: theme.spacing.lg, color: theme.colors.secondary, fontSize: theme.fontSize.sm }}>
        {steps.map(function (step) {
          return (
            <li key={step} style={{ marginBottom: theme.spacing.xs }}>
              {step}
            </li>
          );
        })}
      </ol>
    </section>
  );
}

