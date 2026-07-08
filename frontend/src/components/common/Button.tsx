import type { CSSProperties, ReactNode } from "react";
import { theme } from "../../theme";

type ButtonProps = {
  variant?: "primary" | "outline" | "ghost" | "danger" | "dangerOutline";
  loading?: boolean;
  disabled?: boolean;
  onClick?: () => void;
  type?: "button" | "submit" | "reset";
  children: ReactNode;
  fullWidth?: boolean;
};

export function Button({
  variant = "primary",
  loading = false,
  disabled = false,
  onClick,
  type = "button",
  children,
  fullWidth = false,
}: ButtonProps) {
  const base: CSSProperties = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: theme.spacing.sm,
    padding: `${theme.spacing.sm} ${theme.spacing.lg}`,
    borderRadius: theme.radius.full,
    fontSize: theme.fontSize.sm,
    fontWeight: 700,
    cursor: disabled || loading ? "not-allowed" : "pointer",
    opacity: disabled || loading ? 0.6 : 1,
    border: "none",
    width: fullWidth ? "100%" : undefined,
    transition: "background 0.15s",
  };

  const variants: Record<string, CSSProperties> = {
    primary: {
      background: theme.colors.primary,
      color: theme.colors.background,
    },
    outline: {
      background: "transparent",
      color: theme.colors.primary,
      border: `1px solid ${theme.colors.primary}`,
    },
    ghost: {
      background: "transparent",
      color: theme.colors.secondary,
    },
    danger: {
      background: theme.colors.danger,
      color: theme.colors.background,
    },
    dangerOutline: {
      background: "transparent",
      color: theme.colors.danger,
      border: `1px solid ${theme.colors.danger}`,
    },
  };

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled || loading}
      style={{ ...base, ...variants[variant] }}
    >
      {loading ? "..." : children}
    </button>
  );
}
