import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { api } from '../api/client';
import { Avatar, TitleRow } from '../components/TitleBits';
import { EmptyState, ErrorState, LoadingState, PageHeader, ScorePill } from '../components/Ui';

export function ProfilePage() {
  const { id } = useParams();
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const userUUID = id === 'me' ? me.data?.uuid : id;
  const profile = useQuery({
    queryKey: ['profile', userUUID],
    queryFn: () => api.profileRatings(userUUID as string),
    enabled: Boolean(userUUID),
  });

  if (me.isLoading || profile.isLoading) return <LoadingState label="Открываем профиль" />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (profile.isError) return <ErrorState error={profile.error} />;

  const own = userUUID === me.data?.uuid;
  const profileUser = profile.data?.user || me.data;
  const name = profileUser?.first_name || 'Профиль';

  return (
    <>
      <PageHeader title={own ? 'Профиль' : name} subtitle={own ? name : 'Оценки видны только друзьям'} />
      {profileUser ? (
        <section className="profile-head">
          <Avatar name={profileUser.first_name} url={profileUser.photo_url} />
          <div>
            <h2>{profileUser.first_name}</h2>
            {profileUser.username ? <p className="muted">@{profileUser.username}</p> : null}
          </div>
        </section>
      ) : null}
      <section className="stats">
        <div>
          <span className="muted">Оценок</span>
          <b>{profile.data?.stats.count || 0}</b>
        </div>
        <div>
          <span className="muted">Средняя</span>
          <ScorePill value={profile.data?.stats.avg_score || null} />
        </div>
      </section>
      {profile.data?.ratings.length ? (
        <div className="stack">
          {profile.data.ratings.map((rating) => (
            <TitleRow key={`${rating.title.media_type}-${rating.title.tmdb_id}`} title={rating.title} score={rating.avg_score} />
          ))}
        </div>
      ) : (
        <EmptyState title="Оценок нет" text={own ? 'Поставьте первую оценку через поиск.' : 'Профиль пуст или закрыт для не-друзей.'} />
      )}
    </>
  );
}
