import { Home, PencilLine, User } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";
import { theme } from "../../theme";

export function BottomNav() {
  const navigate = useNavigate();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const tab = params.get("tab") ?? "timeline";

  const tabs = [
    { id: "timeline", label: "ホーム", icon: Home, path: "/" },
    { id: "question", label: "問題", icon: PencilLine, path: "/?tab=question" },
    { id: "profile", label: "プロフィール", icon: User, path: "/?tab=profile" },
  ];

  return (
    <nav
      style={{
        position: "fixed",
        bottom: 0,
        left: 0,
        right: 0,
        background: theme.colors.background,
        borderTop: `1px solid ${theme.colors.border}`,
        display: "flex",
        justifyContent: "space-around",
        padding: `${theme.spacing.sm} 0`,
        zIndex: 100,
      }}
    >
      {tabs.map(function ({ id, label, icon: Icon, path }) {
        const active = tab === id;
        return (
          <button
            key={id}
            onClick={function () {
              navigate(path);
            }}
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: theme.spacing.xs,
              background: "none",
              border: "none",
              cursor: "pointer",
              color: active ? theme.colors.primary : theme.colors.secondary,
              padding: theme.spacing.sm,
              fontSize: theme.fontSize.xs,
              fontWeight: active ? 700 : 400,
            }}
          >
            <Icon size={24} strokeWidth={active ? 2.5 : 1.5} />
            {label}
          </button>
        );
      })}
    </nav>
  );
}
