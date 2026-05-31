type Props = {
  title: string;
  description: string;
};

export function EmptyState({ title, description }: Props) {
  return (
    <div className="timeline-state" role="status">
      <span className="timeline-state__icon" aria-hidden="true">
        +
      </span>
      <strong>{title}</strong>
      <span>{description}</span>
    </div>
  );
}
