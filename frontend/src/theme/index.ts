export const theme = {
  colors: {
    primary: "#d7f8e8",
    secondary: "#8a98a8",
    border: "#23313d",
    background: "#070b10",
    backgroundAlt: "#111820",
    danger: "#ff6673",
    success: "#35d399",
  },
  fontSize: {
    xs: "0.75rem",
    sm: "0.875rem",
    base: "1rem",
    lg: "1.125rem",
    xl: "1.25rem",
  },
  radius: {
    sm: "0.375rem",
    md: "0.75rem",
    full: "9999px",
  },
  spacing: {
    xs: "0.25rem",
    sm: "0.5rem",
    md: "1rem",
    lg: "1.5rem",
    xl: "2rem",
  },
} as const;

export type Theme = typeof theme;
