import { Send, UserPlus } from 'lucide-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { Avatar } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';
import { getInviteUserID, haptic, shareInvite } from '../lib/telegram';

export function FriendsPage() {
  const queryClient = useQueryClient();
  const [params] = useSearchParams();
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const friends = useQuery({ queryKey: ['friends'], queryFn: api.friends });
  const requests = useQuery({ queryKey: ['friendRequests'], queryFn: api.friendRequests });
  const inviteFromTelegram = getInviteUserID();
  const inviteFromURL = params.get('invite')?.match(/^u_(\d+)$/)?.[1];
  const invitedUserID = inviteFromTelegram || (inviteFromURL ? Number(inviteFromURL) : null);

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['friends'] });
    queryClient.invalidateQueries({ queryKey: ['friendRequests'] });
  };
  const create = useMutation({
    mutationFn: (userID: number) => api.createFriendRequest(userID),
    onSuccess: () => {
      haptic('success');
      refresh();
    },
  });
  const accept = useMutation({
    mutationFn: (userID: number) => api.acceptFriendRequest(userID),
    onSuccess: refresh,
  });
  const decline = useMutation({
    mutationFn: (userID: number) => api.deleteFriendRequest(userID),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (userID: number) => api.deleteFriend(userID),
    onSuccess: refresh,
  });

  if (friends.isLoading || requests.isLoading || me.isLoading) return <LoadingState label="Загружаем друзей" />;
  if (friends.isError) return <ErrorState error={friends.error} />;
  if (requests.isError) return <ErrorState error={requests.error} />;

  return (
    <>
      <PageHeader
        title="Друзья"
        subtitle="Заявки и круг оценок"
        action={
          me.data ? (
            <button className="icon-button" type="button" onClick={() => shareInvite(me.data.id)} aria-label="Пригласить">
              <Send size={20} />
            </button>
          ) : null
        }
      />
      {invitedUserID && invitedUserID !== me.data?.id ? (
        <section className="panel invite-panel">
          <div>
            <h2>Инвайт в друзья</h2>
            <p>Отправьте заявку пользователю #{invitedUserID}</p>
          </div>
          <Button disabled={create.isPending} onClick={() => create.mutate(invitedUserID)}>
            <UserPlus size={18} /> Добавить
          </Button>
        </section>
      ) : null}
      <section className="panel">
        <div className="panel__header">
          <div>
            <h2>Входящие заявки</h2>
            <p>{requests.data?.requests.length || 'Нет заявок'}</p>
          </div>
        </div>
        {requests.data?.requests.length ? (
          <div className="stack tight">
            {requests.data.requests.map((request) => (
              <div className="user-row" key={request.user.id}>
                <Avatar name={request.user.first_name} url={request.user.photo_url} />
                <span>{request.user.first_name}</span>
                <Button onClick={() => accept.mutate(request.user.id)}>Принять</Button>
                <Button variant="ghost" onClick={() => decline.mutate(request.user.id)}>
                  Отклонить
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <p className="muted">Когда кто-то откроет вашу ссылку, заявка появится здесь.</p>
        )}
      </section>
      <section className="panel">
        <div className="panel__header">
          <div>
            <h2>Мой круг</h2>
            <p>{friends.data?.friends.length || 'Пока никого'}</p>
          </div>
        </div>
        {friends.data?.friends.length ? (
          <div className="stack tight">
            {friends.data.friends.map((friend) => (
              <div className="user-row" key={friend.id}>
                <Avatar name={friend.first_name} url={friend.photo_url} />
                <Link to={`/profile/${friend.id}`}>{friend.first_name}</Link>
                <Button variant="ghost" onClick={() => remove.mutate(friend.id)}>
                  Удалить
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState title="Добавьте друзей" text="Лента и средние оценки появятся после принятия заявок." />
        )}
      </section>
    </>
  );
}
