import { Compass, Search } from 'lucide-react';
import { useDeferredValue, useEffect } from 'react';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { CatalogCard } from '../components/CatalogCard';
import { InfiniteLoad } from '../components/InfiniteLoad';
import { EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';
import { FriendAvatarFilter, WatchTogetherCard } from '../components/WatchTogetherCard';
import { useWatchlistMutation } from '../hooks/useWatchlistMutation';
import { haptic } from '../lib/telegram';
import { availableFriendSelection, normalizeFriendSelection, sameFriendSelection } from '../lib/watchTogether';
import type { CatalogItem, WatchlistMatchItem } from '../types/api';

type ContentTab = 'discover' | 'recommendations' | 'together';
type MediaFilter = 'all' | 'movie' | 'tv';

export function WatchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const watchlist = useWatchlistMutation();
  const query = searchParams.get('q') || '';
  const trimmed = useDeferredValue(query.trim());
  const searching = trimmed.length >= 2;
  const requestedTab = searchParams.get('tab');
  const activeTab: ContentTab = requestedTab === 'recommendations' || requestedTab === 'together' ? requestedTab : 'discover';
  const requestedType = searchParams.get('type');
  const mediaFilter: MediaFilter = requestedType === 'movie' || requestedType === 'tv' ? requestedType : 'all';
  const rawFriendIDs = searchParams.getAll('friend_id');
  const rawFriendIDsKey = rawFriendIDs.join('\u0000');
  const selectedFriendIDs = normalizeFriendSelection(rawFriendIDs);

  const search = useInfiniteQuery({
    queryKey: ['search', trimmed],
    queryFn: ({ pageParam }) => api.search(trimmed, pageParam),
    getNextPageParam: (last) => last.page < last.total_pages ? last.page + 1 : undefined,
    initialPageParam: 1,
    enabled: searching,
  });
  const discover = useInfiniteQuery({
    queryKey: ['discover', mediaFilter],
    queryFn: ({ pageParam }) => api.discover(mediaFilter, pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
    enabled: !searching && activeTab === 'discover',
  });
  const recommendations = useInfiniteQuery({
    queryKey: ['recommendations'],
    queryFn: ({ pageParam }) => api.recommendations(pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
    enabled: !searching && activeTab === 'recommendations',
  });
  const friends = useQuery({
    queryKey: ['friends'],
    queryFn: api.friends,
    enabled: !searching && activeTab === 'together',
  });
  const matches = useInfiniteQuery({
    queryKey: ['watchlistMatches', selectedFriendIDs],
    queryFn: ({ pageParam }) => api.watchlistMatches(selectedFriendIDs, pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
    enabled: !searching && activeTab === 'together' && Boolean(friends.data?.friends.length),
  });

  useEffect(() => {
    if (!friends.data) return;
    const available = new Set(friends.data.friends.map((friend) => friend.uuid));
    const valid = availableFriendSelection(rawFriendIDs, available);
    if (sameFriendSelection(rawFriendIDs, valid)) return;
    const next = new URLSearchParams(searchParams);
    next.delete('friend_id');
    valid.forEach((uuid) => next.append('friend_id', uuid));
    setSearchParams(next, { replace: true });
  }, [friends.data, rawFriendIDsKey, searchParams, setSearchParams]);

  const setParam = (key: string, value?: string) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value);
    else next.delete(key);
    setSearchParams(next, { replace: true });
  };

  const setFriendIDs = (friendIDs: readonly string[]) => {
    const next = new URLSearchParams(searchParams);
    next.delete('friend_id');
    normalizeFriendSelection(friendIDs).forEach((uuid) => next.append('friend_id', uuid));
    setSearchParams(next, { replace: true });
  };

  const toggleFriend = (friendUUID: string) => {
    haptic('light');
    setFriendIDs(selectedFriendIDs.includes(friendUUID)
      ? selectedFriendIDs.filter((uuid) => uuid !== friendUUID)
      : [...selectedFriendIDs, friendUUID]);
  };

  const source = searching ? search : activeTab === 'discover' ? discover : activeTab === 'recommendations' ? recommendations : matches;
  const items: CatalogItem[] = searching
    ? dedupe(search.data?.pages.flatMap((page) => page.results) || [])
    : activeTab === 'discover'
      ? dedupe(discover.data?.pages.flatMap((page) => page.items) || [])
      : activeTab === 'recommendations'
        ? dedupe(recommendations.data?.pages.flatMap((page) => page.items) || [])
        : [];
  const matchItems = dedupeMatches(matches.data?.pages.flatMap((page) => page.items) || []);
  const degraded = !searching && activeTab !== 'together' && (activeTab === 'discover'
    ? discover.data?.pages.some((page) => page.degraded)
    : recommendations.data?.pages.some((page) => page.degraded));
  const personalized = recommendations.data?.pages[0]?.personalized;
  const sourceState = source.isLoading ? 'loading' : source.isError ? 'error' : 'ready';

  return (
    <>
      <PageHeader title="Смотреть" subtitle="Найдите следующее кино или сериал" />
      <label className="search-box">
        <Search size={20} aria-hidden />
        <input
          value={query}
          onChange={(event) => setParam('q', event.target.value || undefined)}
          placeholder="Название фильма или сериала"
          aria-label="Поиск фильмов и сериалов"
        />
      </label>

      {!searching ? (
        <div className="segmented watch-segmented" role="tablist" aria-label="Режим просмотра">
          <button role="tab" aria-selected={activeTab === 'discover'} className={activeTab === 'discover' ? 'is-active' : ''} onClick={() => setParam('tab', 'discover')}>
            Обзор
          </button>
          <button role="tab" aria-selected={activeTab === 'recommendations'} className={activeTab === 'recommendations' ? 'is-active' : ''} onClick={() => setParam('tab', 'recommendations')}>
            Для вас
          </button>
          <button role="tab" aria-selected={activeTab === 'together'} className={activeTab === 'together' ? 'is-active' : ''} onClick={() => setParam('tab', 'together')}>
            Вместе
          </button>
        </div>
      ) : null}

      <div key={activeTab} className="tab-panel-transition">
        {!searching ? (
          <>
            {activeTab === 'discover' ? (
              <div className="filter-chips" aria-label="Тип контента">
                {([['all', 'Всё'], ['movie', 'Фильмы'], ['tv', 'Сериалы']] as const).map(([value, label]) => (
                  <button key={value} className={mediaFilter === value ? 'is-active' : ''} aria-pressed={mediaFilter === value} onClick={() => setParam('type', value)}>{label}</button>
                ))}
              </div>
            ) : null}
            {activeTab === 'recommendations' && personalized === false ? (
              <div className="catalog-notice"><Compass size={17} aria-hidden /> Оцените хотя бы три тайтла — подборка станет персональной.</div>
            ) : null}
            {activeTab === 'together' && friends.isLoading ? <LoadingState label="Загружаем друзей" /> : null}
            {activeTab === 'together' && friends.isError ? <ErrorState error={friends.error} /> : null}
            {activeTab === 'together' && friends.data?.friends.length ? (
              <FriendAvatarFilter
                friends={friends.data.friends}
                selected={selectedFriendIDs}
                onToggle={toggleFriend}
                onClear={() => {
                  haptic('light');
                  setFriendIDs([]);
                }}
              />
            ) : null}
          </>
        ) : null}

        <div key={sourceState} className="async-content-fade">
          {query.trim().length > 0 && query.trim().length < 2 ? <EmptyState title="Введите ещё символ" text="Для поиска нужно минимум два символа." /> : null}
          {source.isLoading ? <LoadingState label={searching ? 'Ищем' : 'Собираем подборку'} /> : null}
          {source.isError ? <ErrorState error={source.error} /> : null}
          {degraded ? <div className="catalog-notice">Часть каталога временно недоступна, показываем оставшиеся результаты.</div> : null}
          {(searching || activeTab !== 'together') && !source.isLoading && !source.isError && items.length ? (
            <div className="stack">
              {items.map((item) => (
                <CatalogCard
                  key={`${item.title.media_type}-${item.title.tmdb_id}`}
                  item={item}
                  pending={watchlist.isPending}
                  onToggle={(next) => watchlist.mutate({ title: item.title, inWatchlist: next })}
                />
              ))}
              <InfiniteLoad hasNext={Boolean(source.hasNextPage)} loading={source.isFetchingNextPage} onLoad={() => source.fetchNextPage()} />
            </div>
          ) : null}
          {!searching && activeTab === 'together' && !matches.isLoading && !matches.isError && matchItems.length ? (
            <div className="stack">
              {matchItems.map((item) => <WatchTogetherCard key={`${item.title.media_type}-${item.title.tmdb_id}`} item={item} />)}
              <InfiniteLoad hasNext={Boolean(matches.hasNextPage)} loading={matches.isFetchingNextPage} onLoad={() => matches.fetchNextPage()} />
            </div>
          ) : null}
          {(searching || activeTab !== 'together') && !source.isLoading && !source.isError && !items.length && (searching || !query.trim()) ? (
            <EmptyState title={searching ? 'Ничего не найдено' : 'Подборка закончилась'} text={searching ? 'Попробуйте другое название.' : 'Загляните сюда немного позже.'} />
          ) : null}
          {!searching && activeTab === 'together' && friends.data?.friends.length === 0 ? (
            <EmptyState title="Добавьте друзей" text="Общие планы на просмотр появятся после принятия заявок." />
          ) : null}
          {!searching && activeTab === 'together' && Boolean(friends.data?.friends.length) && !matches.isLoading && !matches.isError && !matchItems.length ? (
            <EmptyState
              title={selectedFriendIDs.length ? 'Нет общих тайтлов' : 'Пока нет общих планов на просмотр'}
              text={selectedFriendIDs.length ? 'Попробуйте изменить выбор друзей.' : 'Добавляйте фильмы и сериалы в «Хочу посмотреть» — совпадения появятся здесь.'}
            />
          ) : null}
        </div>
      </div>
    </>
  );
}

function dedupe(items: CatalogItem[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const key = `${item.title.media_type}:${item.title.tmdb_id}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function dedupeMatches(items: WatchlistMatchItem[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    const key = `${item.title.media_type}:${item.title.tmdb_id}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}
