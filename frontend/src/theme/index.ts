export const theme = {
  colors: {
    primary: "#000000",
    primaryHover: "#1a1a1a",
    secondary: "#536471",
    border: "#eff3f4",
    background: "#ffffff",
    backgroundAlt: "#f7f9f9",
    danger: "#f4212e",
    success: "#00ba7c",
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
