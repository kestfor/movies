import { Check, Clock, Search, Send, UserCheck, UserPlus } from 'lucide-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { UserLink } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';
import { getInviteUserUUID, haptic, shareInvite } from '../lib/telegram';

export function FriendsPage() {
  const queryClient = useQueryClient();
  const [params] = useSearchParams();
  const [search, setSearch] = useState('');
  const normalizedSearch = search.trim().replace(/^@/, '');
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const friends = useQuery({ queryKey: ['friends'], queryFn: api.friends });
  const requests = useQuery({ queryKey: ['friendRequests'], queryFn: api.friendRequests });
  const userSearch = useQuery({
    queryKey: ['userSearch', normalizedSearch],
    queryFn: () => api.userSearch(normalizedSearch),
    enabled: normalizedSearch.length > 0,
  });
  const inviteFromTelegram = getInviteUserUUID();
  const inviteFromURL = params.get('invite')?.match(/^uid_([0-9a-fA-F-]{36})$/)?.[1] || null;
  const invitedUserUUID = inviteFromTelegram || inviteFromURL;

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ['friends'] });
    queryClient.invalidateQueries({ queryKey: ['friendRequests'] });
    queryClient.invalidateQueries({ queryKey: ['userSearch'] });
  };
  const create = useMutation({
    mutationFn: (userUUID: string) => api.createFriendRequest(userUUID),
    onSuccess: () => {
      haptic('success');
      refresh();
    },
  });
  const accept = useMutation({
    mutationFn: (userUUID: string) => api.acceptFriendRequest(userUUID),
    onSuccess: refresh,
  });
  const decline = useMutation({
    mutationFn: (userUUID: string) => api.deleteFriendRequest(userUUID),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (userUUID: string) => api.deleteFriend(userUUID),
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
            <button className="icon-button" type="button" onClick={() => shareInvite(me.data.uuid)} aria-label="Пригласить">
              <Send size={20} />
            </button>
          ) : null
        }
      />
      {invitedUserUUID && invitedUserUUID !== me.data?.uuid ? (
        <section className="panel invite-panel">
          <div>
            <h2>Инвайт в друзья</h2>
            <p>Отправьте заявку пользователю по приглашению</p>
          </div>
          <Button disabled={create.isPending} onClick={() => create.mutate(invitedUserUUID)}>
            <UserPlus size={18} /> Добавить
          </Button>
        </section>
      ) : null}
      <section className="panel">
        <div className="panel__header">
          <div>
            <h2>Поиск людей</h2>
            <p>По username в Telegram</p>
          </div>
          <Search size={22} />
        </div>
        <label className="search-box">
          <Search size={18} />
          <input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="@username"
            autoCapitalize="none"
            autoCorrect="off"
          />
        </label>
        {userSearch.isError ? <ErrorState error={userSearch.error} /> : null}
        {normalizedSearch && userSearch.isLoading ? <LoadingState label="Ищем людей" /> : null}
        {normalizedSearch && userSearch.data?.users.length ? (
          <div className="stack tight">
            {userSearch.data.users.map((item) => (
              <div className="user-row" key={item.user.uuid}>
                <UserLink user={item.user} />
                {item.relationship === 'self' ? <span className="muted">Это вы</span> : null}
                {item.relationship === 'friend' ? (
                  <Link className="button button--ghost" to={`/profile/${item.user.uuid}`} onClick={() => haptic('light')}>
                    <UserCheck size={18} /> Профиль
                  </Link>
                ) : null}
                {item.relationship === 'incoming' ? (
                  <Button disabled={accept.isPending} onClick={() => accept.mutate(item.user.uuid)}>
                    <Check size={18} /> Принять
                  </Button>
                ) : null}
                {item.relationship === 'outgoing' ? (
                  <span className="status-chip">
                    <Clock size={14} /> Заявка отправлена
                  </span>
                ) : null}
                {item.relationship === 'none' ? (
                  <Button disabled={create.isPending} onClick={() => create.mutate(item.user.uuid)}>
                    <UserPlus size={18} /> Добавить
                  </Button>
                ) : null}
              </div>
            ))}
          </div>
        ) : null}
        {normalizedSearch && !userSearch.isLoading && userSearch.data?.users.length === 0 ? (
          <p className="muted">Никого не нашли.</p>
        ) : null}
      </section>
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
              <div className="user-row" key={request.user.uuid}>
                <UserLink user={request.user} />
                <div className="user-row__actions">
                  <Button onClick={() => accept.mutate(request.user.uuid)}>Принять</Button>
                  <Button variant="ghost" onClick={() => decline.mutate(request.user.uuid)}>
                    Отклонить
                  </Button>
                </div>
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
              <div className="user-row" key={friend.uuid}>
                <UserLink user={friend} />
                <Button variant="ghost" onClick={() => remove.mutate(friend.uuid)}>
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
