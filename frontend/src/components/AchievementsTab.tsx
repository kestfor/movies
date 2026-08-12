import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { EmptyState, ErrorState, LoadingState } from './Ui';

export function AchievementsTab({
  userUUID,
  own,
  visible,
  highlighted,
}: {
  userUUID: string;
  own: boolean;
  visible: boolean;
  highlighted?: string | null;
}) {
  const achievements = useQuery({
    queryKey: ['achievements', 'profile', userUUID],
    queryFn: () => api.achievements(userUUID),
    enabled: visible,
  });

  useEffect(() => {
    if (!highlighted || !achievements.data) return;
    const frame = window.requestAnimationFrame(() => {
      document.getElementById(`achievement-${highlighted}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [achievements.data, highlighted]);

  if (!visible) return <EmptyState title="Ачивки доступны друзьям" text="Добавьте пользователя в друзья, чтобы увидеть достижения." />;
  if (achievements.isLoading) return <LoadingState label="Собираем достижения" />;
  if (achievements.isError) return <ErrorState error={achievements.error} />;
  if (!achievements.data) return <EmptyState title="Достижения недоступны" text="Попробуйте открыть профиль ещё раз." />;

  const { achievements: items } = achievements.data;

  return (
    <section className="achievements-tab" aria-label="Достижения пользователя">
      {own ? <p className="achievement-baseline-note">Прогресс новых достижений считается с момента их появления.</p> : null}
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
