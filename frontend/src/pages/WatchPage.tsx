import { Compass, Search } from 'lucide-react';
import { useDeferredValue } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { CatalogCard } from '../components/CatalogCard';
import { InfiniteLoad } from '../components/InfiniteLoad';
import { EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';
import { useWatchlistMutation } from '../hooks/useWatchlistMutation';
import type { CatalogItem } from '../types/api';

type ContentTab = 'discover' | 'recommendations';
type MediaFilter = 'all' | 'movie' | 'tv';

export function WatchPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const watchlist = useWatchlistMutation();
  const query = searchParams.get('q') || '';
  const trimmed = useDeferredValue(query.trim());
  const searching = trimmed.length >= 2;
  const activeTab: ContentTab = searchParams.get('tab') === 'recommendations' ? 'recommendations' : 'discover';
  const requestedType = searchParams.get('type');
  const mediaFilter: MediaFilter = requestedType === 'movie' || requestedType === 'tv' ? requestedType : 'all';

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

  const setParam = (key: string, value?: string) => {
    const next = new URLSearchParams(searchParams);
    if (value) next.set(key, value);
    else next.delete(key);
    setSearchParams(next, { replace: true });
  };

  const source = searching ? search : activeTab === 'discover' ? discover : recommendations;
  const items = searching
    ? dedupe(search.data?.pages.flatMap((page) => page.results) || [])
    : activeTab === 'discover'
      ? dedupe(discover.data?.pages.flatMap((page) => page.items) || [])
      : dedupe(recommendations.data?.pages.flatMap((page) => page.items) || []);
  const degraded = !searching && (activeTab === 'discover'
    ? discover.data?.pages.some((page) => page.degraded)
    : recommendations.data?.pages.some((page) => page.degraded));
  const personalized = recommendations.data?.pages[0]?.personalized;

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
        <>
          <div className="segmented" role="tablist" aria-label="Режим просмотра">
            <button role="tab" aria-selected={activeTab === 'discover'} className={activeTab === 'discover' ? 'is-active' : ''} onClick={() => setParam('tab', 'discover')}>
              Обзор
            </button>
            <button role="tab" aria-selected={activeTab === 'recommendations'} className={activeTab === 'recommendations' ? 'is-active' : ''} onClick={() => setParam('tab', 'recommendations')}>
              Для вас
            </button>
          </div>
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
        </>
      ) : null}

      {query.trim().length > 0 && query.trim().length < 2 ? <EmptyState title="Введите ещё символ" text="Для поиска нужно минимум два символа." /> : null}
      {source.isLoading ? <LoadingState label={searching ? 'Ищем' : 'Собираем подборку'} /> : null}
      {source.isError ? <ErrorState error={source.error} /> : null}
      {degraded ? <div className="catalog-notice">Часть каталога временно недоступна, показываем оставшиеся результаты.</div> : null}
      {!source.isLoading && !source.isError && items.length ? (
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
      {!source.isLoading && !source.isError && !items.length && (searching || !query.trim()) ? (
        <EmptyState title={searching ? 'Ничего не найдено' : 'Подборка закончилась'} text={searching ? 'Попробуйте другое название.' : 'Загляните сюда немного позже.'} />
      ) : null}
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
