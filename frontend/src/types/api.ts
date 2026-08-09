export type MediaType = 'movie' | 'tv';

export type User = {
  uuid: string;
  username?: string;
  first_name: string;
  photo_url?: string;
  created_at?: string;
};

export type Title = {
  id?: number;
  tmdb_id: number;
  media_type: MediaType;
  title: string;
  original_title?: string;
  release_year?: number;
  poster_path?: string;
  genres?: string[];
  overview?: string;
};

export type Criterion = {
  id: number;
  code: string;
  name: string;
  description: string;
  sort_order: number;
};

export type RatingWithUser = {
  user: User;
  avg_score: number;
  scores: Record<string, number>;
  created_at: string;
  updated_at?: string;
};

export type Rating = {
  id: number;
  title_id: number;
  avg_score: number;
  scores: Record<string, number>;
  created_at: string;
  updated_at?: string;
};

export type FriendsAverage = {
  overall: number;
  by_criteria: Record<string, number>;
};

export type TitleCard = {
  title: Title;
  my_rating: RatingWithUser | null;
  friends_ratings: RatingWithUser[];
  friends_avg: FriendsAverage | null;
  comments_count: number;
  in_watchlist: boolean;
};

export type CatalogItem = {
  title: Title;
  in_watchlist: boolean;
  reason?: string;
};

export type CatalogPage = {
  items: CatalogItem[];
  next_cursor?: string;
  personalized?: boolean;
  degraded: boolean;
};

export type Comment = {
  id: number;
  title_id: number;
  user: User;
  parent_id?: number;
  body: string;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
  replies?: Comment[];
};

export type SearchPage = {
  page: number;
  total_pages: number;
  total_results: number;
  results: CatalogItem[];
};

export type FeedItem = {
  id: number;
  user: User;
  title: Title;
  avg_score: number;
  scores: Record<string, number>;
  created_at: string;
  updated_at?: string;
};

export type FeedPage = {
  items: FeedItem[];
  next_cursor?: string;
};

export type NotificationItem = {
  event_id: number;
  kind: 'rating_created' | 'comment_created' | 'achievement_unlocked';
  actor: User;
  title?: Title;
  rating?: {
    id: number;
    avg_score: number;
  };
  comment?: {
    id: number;
    body: string;
  };
  achievement?: {
    award_id: string;
    title?: string;
    secret: boolean;
  };
  read_at?: string;
  created_at: string;
  deep_link: string;
};

export type Achievement = {
  award_id?: string;
  code?: string;
  title?: string;
  description?: string;
  category?: string;
  icon?: string;
  xp?: number;
  secret: boolean;
  unlocked: boolean;
  sort_order: number;
  earned_at?: string;
  progress?: { value: number; target: number };
};

export type GamificationSummary = {
  total_xp: number;
  level: number;
  rank_title: string;
  current_level_xp: number;
  next_level_xp: number;
  unlocked_count: number;
  total_count: number;
  leaderboard_rank?: number;
};

export type AchievementsPage = {
  user: User;
  relationship: ProfileRatingsPage['relationship'];
  summary: GamificationSummary;
  achievements: Achievement[];
};

export type AchievementLeaderboard = {
  items: Array<{
    rank: number;
    user: User;
    total_xp: number;
    level: number;
    unlocked_count: number;
  }>;
};

export type UnseenAchievements = {
  items: Achievement[];
  backfill_count: number;
  backfill_award_ids?: string[];
};

export type NotificationsPage = {
  items: NotificationItem[];
  next_cursor?: string;
};

export type ProfileRatingsPage = {
  user: User;
  relationship: 'self' | 'friend' | 'incoming' | 'outgoing' | 'none';
  ratings: Array<{
    title: Title;
    avg_score: number;
    scores: Record<string, number>;
    created_at: string;
    updated_at?: string;
  }>;
  stats: {
    count: number;
    avg_score: number;
  };
  next_cursor?: string;
};

export type WatchlistItem = {
  title: Title;
  added_at: string;
};

export type WatchlistPage = {
	items: WatchlistItem[];
	total_count: number;
	next_cursor?: string;
};

export type WatchlistMatchItem = {
  title: Title;
  users: User[];
  matches_count: number;
};

export type WatchlistMatchesPage = {
  items: WatchlistMatchItem[];
  next_cursor?: string;
};

export type FriendRequest = {
  user: User;
  requested_at: string;
};

export type Friendship = {
  status: 'pending' | 'accepted';
  created_at: string;
  responded_at?: string;
};

export type UserSearchResult = {
  user: User;
  relationship: 'self' | 'friend' | 'incoming' | 'outgoing' | 'none';
  can_send_request: boolean;
  can_open_profile: boolean;
  can_accept_request: boolean;
};

export type ApiErrorBody = {
  error?: {
    code: string;
    message: string;
  };
};
