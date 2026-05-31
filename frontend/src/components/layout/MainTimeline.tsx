import type { ReactNode } from "react";

type Props = {
  title: string;
  actions?: ReactNode;
  children: ReactNode;
};

export function MainTimeline({ title, actions, children }: Props) {
  return (
    <section className="main-timeline" aria-labelledby="main-timeline-title">
      <header className="main-timeline__header">
        <h1 id="main-timeline-title">{title}</h1>
        {actions ? <div className="main-timeline__actions">{actions}</div> : null}
      </header>
      {children}
    </section>
  );
}
