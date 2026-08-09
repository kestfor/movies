import { ArrowUpDown, Check, Clock, UserMinus, UserPlus } from 'lucide-react';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { CatalogCard } from '../components/CatalogCard';
import { AchievementsTab } from '../components/AchievementsTab';
import { CircleRanking } from '../components/CircleRanking';
import { InfiniteLoad } from '../components/InfiniteLoad';
import { ProfileHero } from '../components/ProfileHero';
import { TitleRow } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';
import { useWatchlistMutation } from '../hooks/useWatchlistMutation';
import { haptic } from '../lib/telegram';
import type { CatalogItem } from '../types/api';

const sortPresets = {
  newest: { label: 'Сначала новые', sort: 'recent', order: 'desc' },
  oldest: { label: 'Сначала старые', sort: 'recent', order: 'asc' },
  highest: { label: 'Высокая оценка', sort: 'score', order: 'desc' },
  lowest: { label: 'Низкая оценка', sort: 'score', order: 'asc' },
  title: { label: 'По названию', sort: 'title', order: 'asc' },
} as const;

type SortPreset = keyof typeof sortPresets;

export function ProfilePage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const watchlistMutation = useWatchlistMutation();
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const userUUID = id === 'me' ? me.data?.uuid : id;
  const tabParam = searchParams.get('tab');
  const activeTab = tabParam === 'watchlist' || tabParam === 'achievements' ? tabParam : 'ratings';
  const requestedSort = searchParams.get('sort') as SortPreset | null;
  const sortKey: SortPreset = requestedSort && requestedSort in sortPresets ? requestedSort : 'newest';
  const sorting = sortPresets[sortKey];

  const profile = useInfiniteQuery({
    queryKey: ['profile', userUUID, sortKey],
    queryFn: ({ pageParam }) =>
      api.profileRatings(userUUID as string, pageParam, sorting.sort, sorting.order),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
    enabled: Boolean(userUUID),
  });
  const watchlist = useInfiniteQuery({
    queryKey: ['profileWatchlist', userUUID],
    queryFn: ({ pageParam }) => api.watchlist(userUUID as string, pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
    enabled: Boolean(userUUID),
  });
  const firstPage = profile.data?.pages[0];
  const own = Boolean(userUUID && userUUID === me.data?.uuid);
  const relationship = firstPage?.relationship;
  const gamificationVisible = Boolean(userUUID && (own || relationship === 'friend'));
  const achievements = useQuery({
    queryKey: ['achievements', 'profile', userUUID],
    queryFn: () => api.achievements(userUUID as string),
    enabled: gamificationVisible,
    retry: false,
  });
  const leaderboard = useQuery({
    queryKey: ['achievements', 'leaderboard'],
    queryFn: api.achievementLeaderboard,
    enabled: gamificationVisible,
    retry: false,
  });

  const refreshSocial = () => {
    queryClient.invalidateQueries({ queryKey: ['profile', userUUID] });
    queryClient.invalidateQueries({ queryKey: ['profileWatchlist', userUUID] });
    queryClient.invalidateQueries({ queryKey: ['friends'] });
    queryClient.invalidateQueries({ queryKey: ['friendRequests'] });
    queryClient.invalidateQueries({ queryKey: ['userSearch'] });
    queryClient.invalidateQueries({ queryKey: ['achievements'] });
  };
  const create = useMutation({
    mutationFn: (uuid: string) => api.createFriendRequest(uuid),
    onSuccess: () => {
      haptic('success');
      refreshSocial();
    },
    onError: () => haptic('error'),
  });
  const accept = useMutation({
    mutationFn: (uuid: string) => api.acceptFriendRequest(uuid),
    onSuccess: () => {
      haptic('success');
      refreshSocial();
    },
    onError: () => haptic('error'),
  });
  const remove = useMutation({
    mutationFn: (uuid: string) => api.deleteFriend(uuid),
    onSuccess: refreshSocial,
    onError: () => haptic('error'),
  });

  if (me.isLoading || profile.isLoading) return <LoadingState label="Открываем профиль" />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (profile.isError) return <ErrorState error={profile.error} />;

  const profileUser = firstPage?.user || me.data;
  const ratings = dedupeRatings(profile.data?.pages.flatMap((page) => page.ratings) || []);
  const watchlistItems = dedupeCatalog(
    watchlist.data?.pages.flatMap((page) =>
      page.items.map((item) => ({ title: item.title, in_watchlist: true })),
    ) || [],
  );
  const watchlistState = watchlist.isLoading ? 'loading' : watchlist.isError ? 'error' : 'ready';
  const gamificationState = !gamificationVisible
    ? 'locked'
    : achievements.isError
      ? 'error'
      : achievements.data
        ? 'ready'
        : 'loading';
  const leaderboardState = !gamificationVisible
    ? 'locked'
    : leaderboard.isError
      ? 'error'
      : leaderboard.data
        ? 'ready'
        : 'loading';

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams);
    next.set(key, value);
    setSearchParams(next, { replace: true });
  };

  return (
    <>
      <PageHeader title="Профиль" />
      {profileUser ? (
        <ProfileHero
          user={profileUser}
          stats={firstPage?.stats}
          statsVisible={gamificationVisible}
          summary={achievements.data?.summary}
          gamificationState={gamificationState}
          action={firstPage?.relationship === 'none' ? (
            <Button disabled={create.isPending} onClick={() => create.mutate(profileUser.uuid)}>
              <UserPlus size={18} /> Добавить
            </Button>
          ) : firstPage?.relationship === 'incoming' ? (
            <Button disabled={accept.isPending} onClick={() => accept.mutate(profileUser.uuid)}>
              <Check size={18} /> Принять
            </Button>
          ) : firstPage?.relationship === 'outgoing' ? (
            <span className="status-chip"><Clock size={14} /> Заявка отправлена</span>
          ) : firstPage?.relationship === 'friend' ? (
            <Button
              variant="ghost"
              disabled={remove.isPending}
              onClick={() => {
                haptic('warning');
                remove.mutate(profileUser.uuid);
              }}
            >
              <UserMinus size={18} /> Удалить
            </Button>
          ) : undefined}
        />
      ) : null}
      {userUUID ? (
        <CircleRanking
          key={userUUID}
          data={leaderboard.data}
          targetUUID={userUUID}
          state={leaderboardState}
          onRetry={() => leaderboard.refetch()}
        />
      ) : null}

      <div className="segmented profile-segmented" role="tablist" aria-label="Разделы профиля">
        <button className={activeTab === 'ratings' ? 'is-active' : ''} role="tab" aria-selected={activeTab === 'ratings'} onClick={() => setParam('tab', 'ratings')}>
          Оценки
        </button>
        <button className={activeTab === 'watchlist' ? 'is-active' : ''} role="tab" aria-selected={activeTab === 'watchlist'} onClick={() => setParam('tab', 'watchlist')}>
          <span className="profile-tab-label profile-tab-label--full">Хочу посмотреть</span>
          <span className="profile-tab-label profile-tab-label--short">Список</span>
        </button>
        <button className={activeTab === 'achievements' ? 'is-active' : ''} role="tab" aria-selected={activeTab === 'achievements'} onClick={() => setParam('tab', 'achievements')}>
          Ачивки
        </button>
      </div>

      <div key={activeTab} className="tab-panel-transition">
        {activeTab === 'ratings' ? (
          <section aria-label="Оценки пользователя">
            <div className="sort-toolbar">
              <span className="sort-toolbar__label">{sorting.label}</span>
              <label className="sort-control" title="Изменить сортировку">
                <ArrowUpDown size={16} aria-hidden />
                <select aria-label={`Сортировка оценок: ${sorting.label}`} value={sortKey} onChange={(event) => setParam('sort', event.target.value)}>
                  {Object.entries(sortPresets).map(([value, preset]) => <option key={value} value={value}>{preset.label}</option>)}
                </select>
              </label>
            </div>
            {ratings.length ? (
              <div className="stack">
                {ratings.map((rating) => <TitleRow key={`${rating.title.media_type}-${rating.title.tmdb_id}`} title={rating.title} score={rating.avg_score} />)}
                <InfiniteLoad hasNext={Boolean(profile.hasNextPage)} loading={profile.isFetchingNextPage} onLoad={() => profile.fetchNextPage()} />
              </div>
            ) : (
              <EmptyState
                title="Оценок нет"
                text={own ? 'Поставьте первую оценку на экране «Смотреть».' : gamificationVisible ? 'Пользователь пока ничего не оценил.' : 'Оценки доступны только друзьям.'}
              />
            )}
          </section>
        ) : activeTab === 'watchlist' ? (
          <section aria-label="Хочу посмотреть">
            <div key={watchlistState} className="async-content-fade">
              {watchlist.isLoading ? <LoadingState label="Загружаем список" /> : null}
              {watchlist.isError ? <ErrorState error={watchlist.error} /> : null}
              {!watchlist.isLoading && !watchlist.isError && watchlistItems.length ? (
                <div className="stack">
                  {watchlistItems.map((item) => (
                    <CatalogCard
                      key={`${item.title.media_type}-${item.title.tmdb_id}`}
                      item={item}
                      writable={own}
                      pending={watchlistMutation.isPending}
                      onToggle={(next) => watchlistMutation.mutate({ title: item.title, inWatchlist: next })}
                    />
                  ))}
                  <InfiniteLoad hasNext={Boolean(watchlist.hasNextPage)} loading={watchlist.isFetchingNextPage} onLoad={() => watchlist.fetchNextPage()} />
                </div>
              ) : null}
              {!watchlist.isLoading && !watchlist.isError && !watchlistItems.length ? (
                <EmptyState
                  title="Список пуст"
                  text={own ? 'Добавляйте фильмы и сериалы на экране «Смотреть».' : gamificationVisible ? 'Пользователь пока ничего не добавил.' : 'Список доступен только друзьям.'}
                />
              ) : null}
            </div>
          </section>
        ) : userUUID ? (
          <AchievementsTab userUUID={userUUID} own={own} visible={gamificationVisible} highlighted={searchParams.get('achievement')} />
        ) : null}
      </div>
    </>
  );
}

function dedupeRatings<T extends { title: { media_type: string; tmdb_id: number } }>(items: T[]) {
  return dedupeByTitle(items, (item) => item.title);
}

function dedupeCatalog(items: CatalogItem[]) {
  return dedupeByTitle(items, (item) => item.title);
}

function dedupeByTitle<T>(items: T[], titleOf: (item: T) => { media_type: string; tmdb_id: number }) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const title = titleOf(item);
    const key = `${title.media_type}:${title.tmdb_id}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}
