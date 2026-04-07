import type { ReactNode } from "react";
import { BottomNav } from "./BottomNav";

type Props = { children: ReactNode };

export function AppLayout({ children }: Props) {
  return (
    <div
      style={{
        maxWidth: "480px",
        margin: "0 auto",
        minHeight: "100vh",
        paddingBottom: "5rem",
        position: "relative",
      }}
    >
      {children}
      <BottomNav />
    </div>
  );
}
