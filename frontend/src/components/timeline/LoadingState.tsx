import { Spinner } from "../common/Spinner";

export function LoadingState() {
  return (
    <div className="timeline-state" role="status" aria-label="読み込み中">
      <Spinner />
    </div>
  );
}
