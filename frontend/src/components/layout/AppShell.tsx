import type { ReactNode } from "react";
import { KindleSyncBootstrap } from "../system/KindleSyncBootstrap";
import { BottomTabBar } from "./BottomTabBar";
import { LeftSidebar } from "./LeftSidebar";
import { RightSidebar } from "./RightSidebar";

type Props = {
  children: ReactNode;
};

export function AppShell({ children }: Props) {
  return (
    <div className="app-shell">
      <KindleSyncBootstrap />
      <LeftSidebar />
      <main className="app-shell__main" aria-label="学習タイムライン">
        {children}
      </main>
      <RightSidebar />
      <BottomTabBar />
    </div>
  );
}
