export const colors = {
  primary: "#1DA1F2",
  dark: "#1a91da",
  background: {
    default: "#000000",
    secondary: "#16181C",
    tertiary: "#202327",
  },
  border: "#2F3336",
  text: {
    primary: "#E7E9EA",
    secondary: "#71767B",
    muted: "#3D4144",
  },
} as const;

export const fontFamily = {
  sans: ["Inter", "Noto Sans JP", "system-ui", "sans-serif"],
} as const;

export const borderRadius = {
  card: "12px",
} as const;

export const boxShadow = {
  card: "0 4px 24px rgba(255, 255, 255, 0.04)",
  modal: "0 16px 48px rgba(0, 0, 0, 0.32)",
} as const;

export const theme = {
  colors,
  fontFamily,
  borderRadius,
  boxShadow,
} as const;

export type Theme = typeof theme;
