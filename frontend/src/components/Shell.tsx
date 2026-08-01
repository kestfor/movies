import { Compass, Film, Users, UserRound } from 'lucide-react';
import { NavLink, Outlet, useLocation, useNavigate, useNavigationType } from 'react-router-dom';
import { useEffect, useRef } from 'react';
import { flushSync } from 'react-dom';
import { haptic, tg } from '../lib/telegram';
import {
  clearActiveTitleTransition,
  consumePreserveScroll,
  consumeSuppressPageTransition,
  getActiveTitleTransitionByRef,
  preserveNextScroll,
  startViewTransition,
  suppressNextPageTransition,
  titleTransitionNameFromRef,
} from '../lib/transitions';

const tabs = [
  { to: '/feed', label: 'Лента', icon: Film },
  { to: '/watch', label: 'Смотреть', icon: Compass },
  { to: '/friends', label: 'Друзья', icon: Users },
  { to: '/profile/me', label: 'Профиль', icon: UserRound },
];

export function Shell() {
  const location = useLocation();
  const navigate = useNavigate();
  const navigationType = useNavigationType();
  const routeState = location.state as { pageDirection?: string } | null;
  const pageDirection = routeState?.pageDirection || (navigationType === 'POP' ? 'back' : 'forward');
  const didMount = useRef(false);
  const suppressPageTransition = consumeSuppressPageTransition();

  useEffect(() => {
    didMount.current = true;
  }, []);

  useEffect(() => {
    if (!didMount.current) return;
    if (consumePreserveScroll()) return;
    window.scrollTo(0, 0);
  }, [location.pathname]);

  useEffect(() => {
    const back = () => {
      haptic('light');
      if (runTitleBackTransition(location.pathname, navigate)) return;
      navigate(-1);
    };
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
        <div
          key={location.pathname}
          className={
            didMount.current && !suppressPageTransition
              ? `page-transition page-transition--${pageDirection}`
              : 'page-transition'
          }
        >
          <Outlet />
        </div>
      </main>
      <nav className="tabbar" aria-label="Основная навигация">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          const currentIndex = tabIndexForPath(location.pathname);
          const nextIndex = tabs.findIndex((item) => item.to === tab.to);
          const direction = currentIndex >= 0 && nextIndex > currentIndex ? 'tab-forward' : 'tab-back';
          return (
            <NavLink
              key={tab.to}
              to={tab.to}
              className="tabbar__item"
              onClick={(event) => {
                if (currentIndex === nextIndex) return;
                haptic('light');
                event.preventDefault();
                navigate(tab.to, { state: { pageDirection: direction } });
              }}
            >
              <Icon size={20} aria-hidden />
              <span>{tab.label}</span>
            </NavLink>
          );
        })}
      </nav>
    </div>
  );
}

function tabIndexForPath(pathname: string) {
  if (pathname === '/feed') return 0;
  if (pathname.startsWith('/watch') || pathname.startsWith('/search')) return 1;
  if (pathname.startsWith('/friends')) return 2;
  if (pathname.startsWith('/profile')) return 3;
  return -1;
}

function runTitleBackTransition(pathname: string, navigate: ReturnType<typeof useNavigate>) {
  const match = pathname.match(/^\/title\/([^/]+)\/([^/]+)/);
  if (!match) return false;

  const [, mediaType, tmdbID] = match;
  const active = getActiveTitleTransitionByRef(mediaType, tmdbID);
  if (!active) return false;

  const hero = document.querySelector<HTMLElement>('.title-hero');
  const transitionName = titleTransitionNameFromRef(mediaType, tmdbID);
  hero?.style.setProperty('view-transition-name', transitionName);

  let target: HTMLElement | undefined;
  const transition = startViewTransition(() => {
    suppressNextPageTransition();
    preserveNextScroll();
    flushSync(() => navigate(-1));
    window.scrollTo(0, active.sourceScrollY);
    target = findTitleRow(active.ref);
    target?.style.setProperty('view-transition-name', transitionName);
  });

  if (!transition) {
    hero?.style.removeProperty('view-transition-name');
    clearActiveTitleTransition();
    return true;
  }

  transition.finished.finally(() => {
    hero?.style.removeProperty('view-transition-name');
    target?.style.removeProperty('view-transition-name');
    clearActiveTitleTransition();
  });
  return true;
}

function findTitleRow(ref: string) {
  return Array.from(document.querySelectorAll<HTMLElement>('[data-title-transition-id]')).find(
    (element) => element.dataset.titleTransitionId === ref,
  );
}
