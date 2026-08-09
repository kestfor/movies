import { getInitData } from '../lib/telegram';
import type {
  ApiErrorBody,
  AchievementLeaderboard,
  AchievementsPage,
  Comment,
  Criterion,
  CatalogPage,
  FeedPage,
  FriendRequest,
  Friendship,
  NotificationsPage,
  ProfileRatingsPage,
  Rating,
  SearchPage,
  TitleCard,
  User,
  UserSearchResult,
  UnseenAchievements,
  WatchlistPage,
  WatchlistMatchesPage,
} from '../types/api';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const initData = await getInitData();
  const headers = new Headers(init.headers);
  if (init.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (initData) {
    headers.set('Authorization', `tma ${initData}`);
  }

  const response = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (response.status === 204) {
    return undefined as T;
  }

  const text = await response.text();
  const data = text ? JSON.parse(text) : null;

  if (!response.ok) {
    const body = data as ApiErrorBody | null;
    throw new ApiError(
      response.status,
      body?.error?.code || 'request_failed',
      body?.error?.message || 'Запрос не удался',
    );
  }

  return data as T;
}

const qs = (params: Record<string, string | number | readonly string[] | undefined>) => {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((item) => query.append(key, item));
    } else if (value !== undefined && value !== '') {
      query.set(key, String(value));
    }
  });
  const raw = query.toString();
  return raw ? `?${raw}` : '';
};

export const api = {
  me: () => request<User>('/me'),
  userSearch: (q: string) => request<{ users: UserSearchResult[] }>(`/users/search${qs({ q })}`),
  criteria: () => request<{ criteria: Criterion[] }>('/criteria'),
  search: (q: string, page = 1) => request<SearchPage>(`/search${qs({ q, page })}`),
  title: (type: string, id: string | number) => request<TitleCard>(`/titles/${type}/${id}`),
  putRating: (type: string, id: string | number, scores: Record<string, number>) =>
    request<{ rating: Rating; in_watchlist: false }>(`/titles/${type}/${id}/rating`, {
      method: 'PUT',
      body: JSON.stringify({ scores }),
    }),
  deleteRating: (type: string, id: string | number) =>
    request<void>(`/titles/${type}/${id}/rating`, { method: 'DELETE' }),
  comments: (type: string, id: string | number) =>
    request<{ comments: Comment[] }>(`/titles/${type}/${id}/comments`),
  postComment: (type: string, id: string | number, body: string, parentID?: number) =>
    request<{ comment: Comment }>(`/titles/${type}/${id}/comments`, {
      method: 'POST',
      body: JSON.stringify({ body, parent_id: parentID || 0 }),
    }),
  patchComment: (id: number, body: string) =>
    request<{ comment: Comment }>(`/comments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify({ body }),
    }),
  deleteComment: (id: number) =>
    request<{ comment: Comment }>(`/comments/${id}`, { method: 'DELETE' }),
  friends: () => request<{ friends: User[] }>('/friends'),
  friendRequests: () => request<{ requests: FriendRequest[] }>('/friends/requests'),
  createFriendRequest: (userUUID: string) =>
    request<{ friendship: Friendship }>('/friends/requests', {
      method: 'POST',
      body: JSON.stringify({ user_uuid: userUUID }),
    }),
  acceptFriendRequest: (userUUID: string) =>
    request<{ friendship: Friendship }>(`/friends/requests/${userUUID}/accept`, { method: 'POST' }),
  deleteFriendRequest: (userUUID: string) =>
    request<void>(`/friends/requests/${userUUID}`, { method: 'DELETE' }),
  deleteFriend: (userUUID: string) => request<void>(`/friends/${userUUID}`, { method: 'DELETE' }),
  feed: (cursor?: string, limit = 20) => request<FeedPage>(`/feed${qs({ cursor, limit })}`),
  notifications: (cursor?: string, limit = 20) => request<NotificationsPage>(`/notifications${qs({ cursor, limit })}`),
  unreadNotificationsCount: () => request<{ count: number }>('/notifications/unread-count'),
  markNotificationRead: (eventID: number) =>
    request<void>(`/notifications/${eventID}/read`, { method: 'POST' }),
  markAllNotificationsRead: () => request<void>('/notifications/read-all', { method: 'POST' }),
  profileRatings: (
    userUUID: string,
    cursor?: string,
    sort?: 'recent' | 'score' | 'title',
    order?: 'asc' | 'desc',
    limit = 20,
  ) => request<ProfileRatingsPage>(`/users/${userUUID}/ratings${qs({ cursor, sort, order, limit })}`),
  watchlist: (userUUID: string, cursor?: string, limit = 20) =>
    request<WatchlistPage>(`/users/${userUUID}/watchlist${qs({ cursor, limit })}`),
  watchlistMatches: (friendUUIDs: readonly string[], cursor?: string, limit = 20) =>
    request<WatchlistMatchesPage>(`/watchlist/matches${qs({ friend_id: friendUUIDs, cursor, limit })}`),
  addToWatchlist: (type: string, id: string | number) =>
    request<{ in_watchlist: true }>(`/titles/${type}/${id}/watchlist`, { method: 'PUT' }),
  removeFromWatchlist: (type: string, id: string | number) =>
    request<void>(`/titles/${type}/${id}/watchlist`, { method: 'DELETE' }),
  discover: (type: 'all' | 'movie' | 'tv', cursor?: string, limit = 20) =>
    request<CatalogPage>(`/discover${qs({ type, cursor, limit })}`),
  recommendations: (cursor?: string, limit = 20) =>
    request<CatalogPage>(`/recommendations${qs({ cursor, limit })}`),
  achievements: (userUUID: string) => request<AchievementsPage>(`/users/${userUUID}/achievements`),
  achievementLeaderboard: () => request<AchievementLeaderboard>('/achievements/leaderboard'),
  unseenAchievements: () => request<UnseenAchievements>('/achievements/unseen'),
  markAchievementsSeen: (awardIDs: string[]) =>
    request<void>('/achievements/seen', {
      method: 'POST',
      body: JSON.stringify({ award_ids: awardIDs }),
    }),
};
