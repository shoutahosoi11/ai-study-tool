import { useLocation, useNavigate } from "react-router-dom";
import { navItems } from "./navigation";

function currentTab(search: string) {
  const params = new URLSearchParams(search);
  return params.get("tab") ?? "timeline";
}

export function BottomTabBar() {
  const navigate = useNavigate();
  const location = useLocation();
  const activeTab = currentTab(location.search);

  return (
    <nav className="bottom-tab-bar" aria-label="モバイルナビゲーション">
      {navItems.map(function ({ id, label, path, icon: Icon }) {
        const active = activeTab === id;
        return (
          <button
            type="button"
            key={id}
            className={active ? "bottom-tab-bar__item bottom-tab-bar__item--active" : "bottom-tab-bar__item"}
            aria-label={label}
            aria-current={active ? "page" : undefined}
            title={label}
            onClick={function () {
              navigate(path);
            }}
          >
            <Icon size={24} strokeWidth={active ? 2.6 : 1.7} aria-hidden="true" />
          </button>
        );
      })}
    </nav>
  );
}
