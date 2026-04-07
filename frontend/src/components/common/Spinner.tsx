import { theme } from "../../theme";

export function Spinner() {
  return (
    <div
      style={{
        width: "1.5rem",
        height: "1.5rem",
        border: `2px solid ${theme.colors.border}`,
        borderTopColor: theme.colors.primary,
        borderRadius: theme.radius.full,
        animation: "spin 0.7s linear infinite",
        display: "inline-block",
      }}
    />
  );
}
