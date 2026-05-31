import { useLocation, useNavigate } from "react-router-dom";
import { navItems } from "./navigation";

function currentTab(search: string) {
  const params = new URLSearchParams(search);
  return params.get("tab") ?? "timeline";
}

export function LeftSidebar() {
  const navigate = useNavigate();
  const location = useLocation();
  const activeTab = currentTab(location.search);

  return (
    <aside className="left-sidebar" aria-label="メインナビゲーション">
      <nav className="left-sidebar__nav">
        <button
          type="button"
          className="left-sidebar__brand"
          aria-label="ホーム"
          title="ホーム"
          onClick={function () {
            navigate("/");
          }}
        >
          <span aria-hidden="true">ai</span>
        </button>
        {navItems.map(function ({ id, label, shortLabel, path, icon: Icon }) {
          const active = activeTab === id;
          return (
            <button
              type="button"
              key={id}
              className={active ? "left-sidebar__item left-sidebar__item--active" : "left-sidebar__item"}
              aria-label={label}
              aria-current={active ? "page" : undefined}
              title={label}
              onClick={function () {
                navigate(path);
              }}
            >
              <Icon size={26} strokeWidth={active ? 2.7 : 1.8} aria-hidden="true" />
              <span className="left-sidebar__label">{shortLabel}</span>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
