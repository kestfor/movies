import { Check, Clock, UserMinus, UserPlus } from 'lucide-react';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { CatalogCard } from '../components/CatalogCard';
import { InfiniteLoad } from '../components/InfiniteLoad';
import { Avatar, TitleRow } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader, ScorePill } from '../components/Ui';
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
  const activeTab = searchParams.get('tab') === 'watchlist' ? 'watchlist' : 'ratings';
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
    enabled: Boolean(userUUID) && activeTab === 'watchlist',
  });

  const refreshSocial = () => {
    queryClient.invalidateQueries({ queryKey: ['profile', userUUID] });
    queryClient.invalidateQueries({ queryKey: ['profileWatchlist', userUUID] });
    queryClient.invalidateQueries({ queryKey: ['friends'] });
    queryClient.invalidateQueries({ queryKey: ['friendRequests'] });
    queryClient.invalidateQueries({ queryKey: ['userSearch'] });
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

  const firstPage = profile.data?.pages[0];
  const own = userUUID === me.data?.uuid;
  const profileUser = firstPage?.user || me.data;
  const name = profileUser?.first_name || 'Профиль';
  const ratings = dedupeRatings(profile.data?.pages.flatMap((page) => page.ratings) || []);
  const watchlistItems = dedupeCatalog(
    watchlist.data?.pages.flatMap((page) =>
      page.items.map((item) => ({ title: item.title, in_watchlist: true })),
    ) || [],
  );

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams);
    next.set(key, value);
    setSearchParams(next, { replace: true });
  };

  return (
    <>
      <PageHeader title={own ? 'Профиль' : name} subtitle={own ? name : 'Оценки и список видны только друзьям'} />
      {profileUser ? (
        <section className="profile-head">
          <Avatar name={profileUser.first_name} url={profileUser.photo_url} />
          <div>
            <h2>{profileUser.first_name}</h2>
            {profileUser.username ? <p className="muted">@{profileUser.username}</p> : null}
          </div>
          {firstPage?.relationship === 'none' ? (
            <Button disabled={create.isPending} onClick={() => create.mutate(profileUser.uuid)}>
              <UserPlus size={18} /> Добавить
            </Button>
          ) : null}
          {firstPage?.relationship === 'incoming' ? (
            <Button disabled={accept.isPending} onClick={() => accept.mutate(profileUser.uuid)}>
              <Check size={18} /> Принять
            </Button>
          ) : null}
          {firstPage?.relationship === 'outgoing' ? (
            <span className="status-chip"><Clock size={14} /> Заявка отправлена</span>
          ) : null}
          {firstPage?.relationship === 'friend' ? (
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
          ) : null}
        </section>
      ) : null}
      <section className="stats">
        <div><span className="muted">Оценок</span><b>{firstPage?.stats.count || 0}</b></div>
        <div><span className="muted">Средняя</span><ScorePill value={firstPage?.stats.avg_score || null} /></div>
      </section>

      <div className="segmented" role="tablist" aria-label="Разделы профиля">
        <button className={activeTab === 'ratings' ? 'is-active' : ''} role="tab" aria-selected={activeTab === 'ratings'} onClick={() => setParam('tab', 'ratings')}>
          Оценки <span>{firstPage?.stats.count || 0}</span>
        </button>
        <button className={activeTab === 'watchlist' ? 'is-active' : ''} role="tab" aria-selected={activeTab === 'watchlist'} onClick={() => setParam('tab', 'watchlist')}>
          Хочу посмотреть {watchlist.data?.pages[0] ? <span>{watchlist.data.pages[0].total_count}</span> : null}
        </button>
      </div>

      {activeTab === 'ratings' ? (
        <section aria-label="Оценки пользователя">
          <label className="sort-control">
            <span className="muted">Сортировка</span>
            <select value={sortKey} onChange={(event) => setParam('sort', event.target.value)}>
              {Object.entries(sortPresets).map(([value, preset]) => <option key={value} value={value}>{preset.label}</option>)}
            </select>
          </label>
          {ratings.length ? (
            <div className="stack">
              {ratings.map((rating) => <TitleRow key={`${rating.title.media_type}-${rating.title.tmdb_id}`} title={rating.title} score={rating.avg_score} />)}
              <InfiniteLoad hasNext={Boolean(profile.hasNextPage)} loading={profile.isFetchingNextPage} onLoad={() => profile.fetchNextPage()} />
            </div>
          ) : (
            <EmptyState title="Оценок нет" text={own ? 'Поставьте первую оценку на экране «Смотреть».' : 'Профиль пуст или закрыт для не-друзей.'} />
          )}
        </section>
      ) : (
        <section aria-label="Хочу посмотреть">
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
            <EmptyState title="Список пуст" text={own ? 'Добавляйте фильмы и сериалы на экране «Смотреть».' : 'Друг пока ничего не добавил.'} />
          ) : null}
        </section>
      )}
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
