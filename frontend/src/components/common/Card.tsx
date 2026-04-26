import type { ReactNode } from "react";
import { theme } from "../../theme";

type CardProps = {
  children: ReactNode;
  onClick?: () => void;
  padding?: string;
};

export function Card({ children, onClick, padding }: CardProps) {
  return (
    <div
      onClick={onClick}
      style={{
        background: theme.colors.background,
        border: `1px solid ${theme.colors.border}`,
        borderRadius: theme.radius.md,
        padding: padding ?? theme.spacing.md,
        cursor: onClick ? "pointer" : undefined,
      }}
    >
      {children}
    </div>
  );
}
