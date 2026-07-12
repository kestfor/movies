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
  results: Title[];
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
