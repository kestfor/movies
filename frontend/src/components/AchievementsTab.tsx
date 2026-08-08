import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Trophy } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { Avatar } from './TitleBits';
import { EmptyState, ErrorState, LoadingState } from './Ui';

export function AchievementsTab({ userUUID, own, highlighted }: { userUUID: string; own: boolean; highlighted?: string | null }) {
  const navigate = useNavigate();
  const achievements = useQuery({
    queryKey: ['achievements', 'profile', userUUID],
    queryFn: () => api.achievements(userUUID),
  });
  const leaderboard = useQuery({
    queryKey: ['achievements', 'leaderboard'],
    queryFn: api.achievementLeaderboard,
  });

  useEffect(() => {
    if (!highlighted || !achievements.data) return;
    const frame = window.requestAnimationFrame(() => {
      document.getElementById(`achievement-${highlighted}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [achievements.data, highlighted]);

  if (achievements.isLoading) return <LoadingState label="Собираем достижения" />;
  if (achievements.isError) return <ErrorState error={achievements.error} />;
  if (!achievements.data) return <EmptyState title="Достижения недоступны" text="Попробуйте открыть профиль ещё раз." />;

  const { summary, achievements: items } = achievements.data;
  const levelStart = summary.current_level_xp;
  const levelSize = Math.max(1, summary.next_level_xp - levelStart);
  const levelProgress = Math.min(100, Math.max(0, ((summary.total_xp - levelStart) / levelSize) * 100));

  return (
    <section className="achievements-tab" aria-label="Достижения пользователя">
      <div className="achievement-summary">
        <div className="achievement-summary__level"><Trophy size={22} /><b>{summary.level}</b><span>уровень</span></div>
        <div className="achievement-summary__main">
          <strong>{summary.rank_title}</strong>
          <span>{summary.total_xp} XP · {summary.unlocked_count} из {summary.total_count}</span>
          <div className="achievement-progress" aria-label={`Прогресс уровня ${Math.round(levelProgress)}%`}>
            <i style={{ width: `${levelProgress}%` }} />
          </div>
        </div>
        {summary.leaderboard_rank ? <span className="achievement-summary__rank">#{summary.leaderboard_rank}</span> : null}
      </div>

      {!items.length ? (
        <EmptyState title="Пока без ачивок" text="Совместные оценки и обсуждения помогут открыть первые достижения." />
      ) : (
        <div className="achievement-grid">
          {items.map((item) => {
            const hidden = item.secret && !item.title;
            const progress = item.progress ? Math.min(100, (item.progress.value / item.progress.target) * 100) : 0;
            return (
              <article
                id={item.award_id ? `achievement-${item.award_id}` : undefined}
                key={item.code || item.award_id || item.sort_order}
                className={`achievement-card achievement-card--${achievementRarity(item.xp)} ${item.unlocked ? 'achievement-card--unlocked' : ''} ${item.award_id === highlighted ? 'achievement-card--highlighted' : ''}`}
              >
                <span className="achievement-medallion" aria-hidden>
                  <span>{hidden ? '❔' : item.icon || '🏆'}</span>
                </span>
                <div className="achievement-card__body">
                  <div className="achievement-card__title"><strong>{hidden ? 'Секретная ачивка' : item.title}</strong>{item.xp ? <span>+{item.xp} XP</span> : null}</div>
                  {!hidden && item.description ? <p>{item.description}</p> : <p>Условие откроется после получения.</p>}
                  {own && !item.unlocked && item.progress ? (
                    <div className="achievement-card__progress">
                      <span><i style={{ width: `${progress}%` }} /></span>
                      <small>{item.progress.value} / {item.progress.target}</small>
                    </div>
                  ) : null}
                  {item.unlocked ? <small className="achievement-card__earned">Получено{item.earned_at ? ` · ${new Date(item.earned_at).toLocaleDateString('ru-RU')}` : ''}</small> : null}
                </div>
              </article>
            );
          })}
        </div>
      )}

      {leaderboard.data?.items.length ? (
        <section className="achievement-leaderboard" aria-label="Рейтинг среди друзей">
          <h3>Кинокруг лидеров</h3>
          <div className="stack">
            {leaderboard.data.items.map((entry) => (
              <button key={entry.user.uuid} type="button" onClick={() => navigate(`/profile/${entry.user.uuid}?tab=achievements`)}>
                <b>#{entry.rank}</b>
                <Avatar name={entry.user.first_name} url={entry.user.photo_url} />
                <span><strong>{entry.user.first_name}</strong><small>{entry.unlocked_count} ачивок · уровень {entry.level}</small></span>
                <em>{entry.total_xp} XP</em>
              </button>
            ))}
          </div>
        </section>
      ) : null}
    </section>
  );
}

function achievementRarity(xp = 0) {
  if (xp >= 350) return 'platinum';
  if (xp >= 200) return 'gold';
  if (xp >= 100) return 'silver';
  if (xp >= 50) return 'bronze';
  return 'mystery';
}
