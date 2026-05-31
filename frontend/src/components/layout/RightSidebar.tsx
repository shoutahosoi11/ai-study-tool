const reviewItems = [
  { icon: "✓", label: "Review", value: "Daily" },
  { icon: "☆", label: "Saved", value: "Queue" },
  { icon: "#", label: "Tags", value: "Books" },
];

const tagItems = ["#anki", "#kindle", "#english", "#exam"];

export function RightSidebar() {
  return (
    <aside className="right-sidebar" aria-label="学習サイドバー">
      <section className="right-sidebar__panel" aria-labelledby="today-review-heading">
        <h2 id="today-review-heading" className="right-sidebar__title">
          Today
        </h2>
        <div className="right-sidebar__metric-list">
          {reviewItems.map(function (item) {
            return (
              <div className="right-sidebar__metric" key={item.label}>
                <span className="right-sidebar__metric-icon" aria-hidden="true">
                  {item.icon}
                </span>
                <span>{item.label}</span>
                <strong>{item.value}</strong>
              </div>
            );
          })}
        </div>
      </section>
      <section className="right-sidebar__panel" aria-labelledby="popular-tags-heading">
        <h2 id="popular-tags-heading" className="right-sidebar__title">
          Tags
        </h2>
        <div className="right-sidebar__tags" aria-label="人気タグ">
          {tagItems.map(function (tag) {
            return <span key={tag}>{tag}</span>;
          })}
        </div>
      </section>
    </aside>
  );
}
