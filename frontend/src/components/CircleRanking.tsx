import { useState } from 'react';
import { ChevronDown, Lock, Trophy } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import type { AchievementLeaderboard } from '../types/api';
import { haptic } from '../lib/telegram';
import { Avatar } from './TitleBits';
import { nextRankGap, rankingPreview } from './circleRanking';
import type { ProfileGamificationState } from './ProfileHero';

export function CircleRanking({
  data,
  targetUUID,
  state,
  onRetry,
}: {
  data?: AchievementLeaderboard;
  targetUUID: string;
  state: ProfileGamificationState;
  onRetry: () => void;
}) {
  const navigate = useNavigate();
  const [expanded, setExpanded] = useState(false);
  const items = data?.items || [];
  const target = items.find((item) => item.user.uuid === targetUUID);
  const gap = nextRankGap(items, targetUUID);
  const shown = expanded ? items : rankingPreview(items, targetUUID);

  return (
    <section className={`circle-ranking circle-ranking--${state}`} aria-label="Рейтинг КиноКруга">
      <div className="circle-ranking__header">
        <div>
          <span><Trophy size={16} aria-hidden="true" /> КиноКруг</span>
          <strong>{rankingSummary(target?.rank, gap)}</strong>
        </div>
        {state === 'ready' && items.length > 3 ? (
          <button
            type="button"
            onClick={() => {
              haptic('light');
              setExpanded((value) => !value);
            }}
            aria-expanded={expanded}
          >
            {expanded ? 'Свернуть' : 'Весь рейтинг'}
            <ChevronDown className={expanded ? 'is-open' : ''} size={15} aria-hidden="true" />
          </button>
        ) : null}
      </div>

      {state === 'locked' ? (
        <div className="circle-ranking__message"><Lock size={18} aria-hidden="true" /><span>Рейтинг доступен после добавления в друзья</span></div>
      ) : null}
      {state === 'loading' ? <div className="circle-ranking__message"><span>Загружаем места в КиноКруге…</span></div> : null}
      {state === 'error' ? (
        <div className="circle-ranking__message">
          <span>Не удалось загрузить рейтинг</span>
          <button type="button" onClick={onRetry}>Повторить</button>
        </div>
      ) : null}
      {state === 'ready' && !items.length ? <div className="circle-ranking__message"><span>В КиноКруге пока нет участников</span></div> : null}
      {state === 'ready' && items.length ? (
        <div className={`circle-ranking__list ${expanded ? 'circle-ranking__list--expanded' : ''}`}>
          {shown.map((entry, index) => (
            <button
              className={`${entry.user.uuid === targetUUID ? 'circle-ranking__entry--target' : ''} ${!expanded && index === 3 ? 'circle-ranking__entry--separated' : ''}`}
              key={entry.user.uuid}
              type="button"
              onClick={() => {
                haptic('light');
                navigate(`/profile/${entry.user.uuid}`);
              }}
            >
              <b>#{entry.rank}</b>
              <Avatar name={entry.user.first_name} url={entry.user.photo_url} />
              <span><strong>{entry.user.first_name}</strong><small>Уровень {entry.level} · {entry.unlocked_count} ачивок</small></span>
              <em>{entry.total_xp} XP</em>
            </button>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function rankingSummary(rank?: number, gap?: { rank: number; xp: number } | null) {
  if (!rank) return 'Места среди друзей';
  if (rank === 1) return '#1 · Первое место в вашем КиноКруге';
  if (gap) return `#${rank} · До #${gap.rank} — ${gap.xp} XP`;
  return `#${rank} в вашем КиноКруге`;
}
