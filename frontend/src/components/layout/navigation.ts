import type { ComponentType } from "react";
import type { LucideProps } from "lucide-react";
import { Home, PencilLine, User } from "lucide-react";

export type NavItem = {
  id: "timeline" | "question" | "profile";
  label: string;
  shortLabel: string;
  path: string;
  icon: ComponentType<LucideProps>;
};

export const navItems: NavItem[] = [
  { id: "timeline", label: "ホーム", shortLabel: "Home", path: "/", icon: Home },
  { id: "question", label: "問題", shortLabel: "Quiz", path: "/?tab=question", icon: PencilLine },
  { id: "profile", label: "プロフィール", shortLabel: "Me", path: "/?tab=profile", icon: User },
];
