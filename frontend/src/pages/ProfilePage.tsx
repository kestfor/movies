import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { api } from '../api/client';
import { TitleRow } from '../components/TitleBits';
import { EmptyState, ErrorState, LoadingState, PageHeader, ScorePill } from '../components/Ui';

export function ProfilePage() {
  const { id } = useParams();
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const userID = id === 'me' ? me.data?.id : Number(id);
  const profile = useQuery({
    queryKey: ['profile', userID],
    queryFn: () => api.profileRatings(userID as number),
    enabled: Boolean(userID),
  });

  if (me.isLoading || profile.isLoading) return <LoadingState label="Открываем профиль" />;
  if (me.isError) return <ErrorState error={me.error} />;
  if (profile.isError) return <ErrorState error={profile.error} />;

  const own = userID === me.data?.id;
  const name = own ? me.data?.first_name || 'Профиль' : `Пользователь #${userID}`;

  return (
    <>
      <PageHeader title={own ? 'Профиль' : name} subtitle={own ? name : 'Оценки видны только друзьям'} />
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
