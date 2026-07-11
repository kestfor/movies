import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { FeedCard } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';

export function FeedPage() {
  const feed = useInfiniteQuery({
    queryKey: ['feed'],
    queryFn: ({ pageParam }) => api.feed(pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
  });

  if (feed.isLoading) return <LoadingState label="Загружаем ленту" />;
  if (feed.isError) return <ErrorState error={feed.error} />;

  const items = feed.data?.pages.flatMap((page) => page.items) || [];

  return (
    <>
      <PageHeader title="Лента" subtitle="Оценки друзей" />
      {!items.length ? (
        <EmptyState title="Пока пусто" text="Добавьте друзей или попросите их поставить первые оценки." />
      ) : (
        <div className="stack">
          {items.map((item) => (
            <FeedCard key={item.id} item={item} />
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
