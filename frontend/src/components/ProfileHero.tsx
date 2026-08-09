import type { ReactNode } from 'react';
import { Lock, Sparkles } from 'lucide-react';
import type { GamificationSummary, ProfileRatingsPage, User } from '../types/api';
import { Avatar } from './TitleBits';
import { profileThemeForLevel } from './profileTheme';

export type ProfileGamificationState = 'loading' | 'ready' | 'locked' | 'error';

export function ProfileHero({
  user,
  stats,
  statsVisible,
  summary,
  gamificationState,
  action,
}: {
  user: User;
  stats?: ProfileRatingsPage['stats'];
  statsVisible: boolean;
  summary?: GamificationSummary;
  gamificationState: ProfileGamificationState;
  action?: ReactNode;
}) {
  const theme = profileThemeForLevel(summary?.level || 1);
  const nextTheme = profileThemeForLevel((summary?.level || 1) + 1);
  const progress = levelProgress(summary);
  const ready = gamificationState === 'ready' && Boolean(summary);
  const levelText = ready ? `LVL ${summary?.level}` : gamificationState === 'loading' ? '…' : '🔒';
  const rankText = ready
    ? summary?.rank_title
    : gamificationState === 'loading'
      ? 'Загружаем уровень'
      : gamificationState === 'error'
        ? 'Уровень временно недоступен'
        : 'Уровень доступен друзьям';

  return (
    <section
      className={`profile-hero profile-hero--${ready ? theme.key : 'locked'} profile-hero--step-${ready ? theme.step : 1} profile-hero--next-${nextTheme.key}`}
    >
      <span className="profile-hero__frames" aria-hidden="true"><i /><i /><i /></span>
      <div className="profile-hero__top">
        <div className="profile-hero__avatar">
          <Avatar name={user.first_name} url={user.photo_url} />
          <span>{levelText}</span>
        </div>
        <div className="profile-hero__identity">
          <h2>{user.first_name}</h2>
          {user.username ? <p>@{user.username}</p> : null}
          <strong>{rankText}</strong>
        </div>
        {action ? <div className="profile-hero__action">{action}</div> : null}
      </div>

      <div className="profile-hero__level">
        <div className="profile-hero__level-copy">
          <span><Sparkles size={13} aria-hidden="true" /> Прогресс уровня</span>
          {ready && summary ? <b>До {summary.level + 1} уровня · {Math.max(0, summary.next_level_xp - summary.total_xp)} XP</b> : null}
          {gamificationState === 'locked' ? <b><Lock size={12} aria-hidden="true" /> Доступно друзьям</b> : null}
        </div>
        <div
          className="profile-hero__progress"
          role="progressbar"
          aria-label={ready ? `Прогресс уровня ${Math.round(progress)}%` : rankText}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={ready ? Math.round(progress) : 0}
        >
          <i style={{ width: `${ready ? progress : 0}%` }} />
          <em aria-hidden="true" />
        </div>
        <div className="profile-hero__xp">
          <span>{ready ? `${summary?.total_xp || 0} XP` : '— XP'}</span>
          <span>{ready ? `${summary?.next_level_xp || 0} XP` : '— XP'}</span>
        </div>
      </div>

      <div className="profile-hero__metrics">
        <ProfileMetric label="Оценок" value={statsVisible ? String(stats?.count || 0) : '—'} />
        <ProfileMetric label="Средняя" value={statsVisible && stats?.avg_score ? stats.avg_score.toFixed(1) : '—'} />
        <ProfileMetric label="Ачивок" value={ready ? String(summary?.unlocked_count || 0) : '—'} />
      </div>
    </section>
  );
}

function ProfileMetric({ label, value }: { label: string; value: string }) {
  return <div><b>{value}</b><span>{label}</span></div>;
}

function levelProgress(summary?: GamificationSummary) {
  if (!summary) return 0;
  const size = Math.max(1, summary.next_level_xp - summary.current_level_xp);
  return Math.min(100, Math.max(0, ((summary.total_xp - summary.current_level_xp) / size) * 100));
}
