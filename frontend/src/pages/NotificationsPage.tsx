import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCheck, MessageCircle, Star } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { Avatar, formatRatingDate } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader, ScorePill } from '../components/Ui';
import type { NotificationItem } from '../types/api';

export function NotificationsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const notifications = useInfiniteQuery({
    queryKey: ['notifications', 'list'],
    queryFn: ({ pageParam }) => api.notifications(pageParam),
    getNextPageParam: (last) => last.next_cursor || undefined,
    initialPageParam: undefined as string | undefined,
  });
  const unread = useQuery({ queryKey: ['notifications', 'unread-count'], queryFn: api.unreadNotificationsCount });
  const markRead = useMutation({
    mutationFn: (eventID: number) => api.markNotificationRead(eventID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });
  const markAllRead = useMutation({
    mutationFn: api.markAllNotificationsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  if (notifications.isLoading) return <LoadingState label="Загружаем уведомления" />;
  if (notifications.isError) return <ErrorState error={notifications.error} />;

  const items = notifications.data?.pages.flatMap((page) => page.items) || [];
  const unreadCount = unread.data?.count || 0;

  return (
    <>
      <PageHeader
        title="Уведомления"
        subtitle={unreadCount ? `${unreadCount} новых` : 'Новых нет'}
        action={
          items.length ? (
            <button
              className="icon-button"
              type="button"
              onClick={() => markAllRead.mutate()}
              disabled={markAllRead.isPending}
              aria-label="Отметить всё прочитанным"
            >
              <CheckCheck size={20} aria-hidden />
            </button>
          ) : null
        }
      />
      {!items.length ? (
        <EmptyState title="Пока тихо" text="Когда друзья поставят оценку или напишут комментарий, события появятся здесь." />
      ) : (
        <div className="stack">
          {items.map((item) => (
            <NotificationCard
              key={item.event_id}
              item={item}
              busy={markRead.isPending}
              onOpen={() => {
                const open = () => navigate(item.deep_link, { state: { pageDirection: 'forward' } });
                if (item.read_at) {
                  open();
                  return;
                }
                markRead.mutate(item.event_id, { onSettled: open });
              }}
            />
          ))}
          {notifications.hasNextPage ? (
            <Button variant="ghost" disabled={notifications.isFetchingNextPage} onClick={() => notifications.fetchNextPage()}>
              {notifications.isFetchingNextPage ? 'Загружаем' : 'Показать ещё'}
            </Button>
          ) : null}
        </div>
      )}
    </>
  );
}

function NotificationCard({
  item,
  busy,
  onOpen,
}: {
  item: NotificationItem;
  busy?: boolean;
  onOpen: () => void;
}) {
  const Icon = item.kind === 'comment_created' ? MessageCircle : Star;
  return (
    <button
      className={`notification-card ${item.read_at ? '' : 'notification-card--unread'}`}
      type="button"
      onClick={onOpen}
      disabled={busy}
    >
      <span className="notification-card__icon" aria-hidden>
        <Icon size={18} />
      </span>
      <span className="notification-card__body">
        <span className="notification-card__top">
          <span className="user-link">
            <Avatar name={item.actor.first_name} url={item.actor.photo_url} />
            <span className="user-link__name">{item.actor.first_name}</span>
          </span>
          <time className="rating-date" dateTime={item.created_at}>
            {formatRatingDate(item.created_at)}
          </time>
        </span>
        <span className="notification-card__text">
          {item.kind === 'comment_created' ? 'оставил(а) комментарий к' : 'поставил(а) оценку'} <b>{item.title.title}</b>
        </span>
        {item.comment ? <span className="notification-card__preview">{item.comment.body}</span> : null}
      </span>
      {item.rating ? <ScorePill value={item.rating.avg_score} /> : null}
      {!item.read_at ? <span className="notification-card__dot" aria-hidden /> : null}
    </button>
  );
}
