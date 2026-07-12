import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { FeedCard } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';

export function FeedPage() {
  const navigate = useNavigate();
  const feed = useInfiniteQuery({
    queryKey: ['feed'],
    queryFn: ({ pageParam }) => api.feed(pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
  });
  const criteria = useQuery({ queryKey: ['criteria'], queryFn: api.criteria });
  const unread = useQuery({ queryKey: ['notifications', 'unread-count'], queryFn: api.unreadNotificationsCount });

  if (feed.isLoading || criteria.isLoading) return <LoadingState label="Загружаем ленту" />;
  if (feed.isError) return <ErrorState error={feed.error} />;
  if (criteria.isError) return <ErrorState error={criteria.error} />;

  const items = feed.data?.pages.flatMap((page) => page.items) || [];
  const labels = Object.fromEntries((criteria.data?.criteria || []).map((criterion) => [criterion.code, criterion.name]));

  return (
    <>
      <PageHeader
        title="Лента"
        subtitle="Оценки друзей"
        action={
          <button
            className="icon-button notification-button"
            type="button"
            onClick={() => navigate('/notifications', { state: { pageDirection: 'forward' } })}
            aria-label={`Уведомления${unread.data?.count ? `: ${unread.data.count}` : ''}`}
          >
            <Bell size={20} aria-hidden />
            {unread.data?.count ? <span className="notification-badge">{formatBadgeCount(unread.data.count)}</span> : null}
          </button>
        }
      />
      {!items.length ? (
        <EmptyState title="Пока пусто" text="Добавьте друзей или попросите их поставить первые оценки." />
      ) : (
        <div className="stack">
          {items.map((item) => (
            <FeedCard key={item.id} item={item} labels={labels} />
          ))}
          {feed.hasNextPage ? (
            <Button variant="ghost" disabled={feed.isFetchingNextPage} onClick={() => feed.fetchNextPage()}>
              {feed.isFetchingNextPage ? 'Загружаем' : 'Показать ещё'}
            </Button>
          ) : null}
        </div>
      )}
    </>
  );
}

function formatBadgeCount(count: number) {
  return count > 99 ? '99+' : String(count);
}
