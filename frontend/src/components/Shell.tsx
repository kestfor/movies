import { Film, Search, Users, UserRound } from 'lucide-react';
import { NavLink, Outlet, useLocation, useNavigate, useNavigationType } from 'react-router-dom';
import { useEffect } from 'react';
import { tg } from '../lib/telegram';

const tabs = [
  { to: '/feed', label: 'Лента', icon: Film },
  { to: '/search', label: 'Поиск', icon: Search },
  { to: '/friends', label: 'Друзья', icon: Users },
  { to: '/profile/me', label: 'Профиль', icon: UserRound },
];

export function Shell() {
  const location = useLocation();
  const navigate = useNavigate();
  const navigationType = useNavigationType();
  const pageDirection = navigationType === 'POP' ? 'back' : 'forward';

  useEffect(() => {
    const back = () => navigate(-1);
    if (location.pathname === '/feed') {
      tg?.BackButton?.hide();
      return;
    }
    tg?.BackButton?.show();
    tg?.BackButton?.onClick(back);
    return () => tg?.BackButton?.offClick(back);
  }, [location.pathname, navigate]);

  return (
    <div className="app-shell">
      <main className="content">
        <div key={location.key} className={`page-transition page-transition--${pageDirection}`}>
          <Outlet />
        </div>
      </main>
      <nav className="tabbar" aria-label="Основная навигация">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <NavLink key={tab.to} to={tab.to} className="tabbar__item">
              <Icon size={20} aria-hidden />
              <span>{tab.label}</span>
            </NavLink>
          );
        })}
      </nav>
    </div>
  );
}
