import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: "#1DA1F2",
        dark: "#1a91da",
        background: {
          DEFAULT: "#000000",
          secondary: "#16181C",
          tertiary: "#202327",
        },
        border: "#2F3336",
        text: {
          primary: "#E7E9EA",
          secondary: "#71767B",
          muted: "#3D4144",
        },
      },
      fontFamily: {
        sans: ["Inter", "Noto Sans JP", "system-ui", "sans-serif"],
      },
      borderRadius: {
        card: "12px",
      },
      boxShadow: {
        card: "0 4px 24px rgba(255, 255, 255, 0.04)",
        modal: "0 16px 48px rgba(0, 0, 0, 0.32)",
      },
    },
  },
  plugins: [],
};

export default config;
