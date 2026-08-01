import { Navigate, Route, Routes } from 'react-router-dom';
import { AuthGate } from './pages/AuthGate';
import { FeedPage } from './pages/FeedPage';
import { FriendsPage } from './pages/FriendsPage';
import { NotificationsPage } from './pages/NotificationsPage';
import { ProfilePage } from './pages/ProfilePage';
import { WatchPage } from './pages/WatchPage';
import { TitlePage } from './pages/TitlePage';
import { Shell } from './components/Shell';

export function App() {
  return (
    <Routes>
      <Route element={<AuthGate />}>
        <Route element={<Shell />}>
          <Route path="/feed" element={<FeedPage />} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/watch" element={<WatchPage />} />
          <Route path="/search" element={<Navigate to="/watch" replace />} />
          <Route path="/title/:type/:id" element={<TitlePage />} />
          <Route path="/friends" element={<FriendsPage />} />
          <Route path="/profile/:id" element={<ProfilePage />} />
          <Route path="*" element={<Navigate to="/feed" replace />} />
        </Route>
      </Route>
    </Routes>
  );
}
