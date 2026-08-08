import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Trophy } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import { haptic } from '../lib/telegram';

type UnlockNotice = {
  ids: string[];
  awardID?: string;
  title: string;
  text: string;
  icon: string;
  xp: number;
};

export function AchievementNotifier() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [active, setActive] = useState<UnlockNotice | null>(null);
  const unseen = useQuery({
    queryKey: ['achievements', 'unseen'],
    queryFn: api.unseenAchievements,
    refetchInterval: active ? false : 15_000,
    retry: false,
  });
  const markSeen = useMutation({
    mutationFn: api.markAchievementsSeen,
    onSettled: () => {
      setActive(null);
      queryClient.invalidateQueries({ queryKey: ['achievements'] });
    },
  });

  useEffect(() => {
    if (active || !unseen.data) return;
    const achievement = unseen.data.items.find((item) => item.award_id);
    if (achievement?.award_id) {
      setActive({
        ids: [achievement.award_id],
        awardID: achievement.award_id,
        title: achievement.title || 'Секретное достижение',
        text: `Получено · +${achievement.xp || 0} XP`,
        icon: achievement.icon || '🏆',
        xp: achievement.xp || 0,
      });
      haptic('success');
      return;
    }
    const ids = unseen.data.backfill_award_ids || [];
    if (unseen.data.backfill_count > 0 && ids.length) {
      setActive({
        ids,
        title: 'Найдены достижения',
        text: `${unseen.data.backfill_count} уже были вами получены`,
        icon: '🎉',
        xp: 200,
      });
      haptic('success');
    }
  }, [active, unseen.data]);

  useEffect(() => {
    if (!active || markSeen.isPending) return;
    const timer = window.setTimeout(() => markSeen.mutate(active.ids), 4200);
    return () => window.clearTimeout(timer);
  }, [active, markSeen]);

  if (!active) return null;

  const open = () => {
    if (!markSeen.isPending) markSeen.mutate(active.ids);
    const achievement = active.awardID ? `&achievement=${active.awardID}` : '';
    navigate(`/profile/me?tab=achievements${achievement}`, { state: { pageDirection: 'forward' } });
  };

  return (
    <div className="achievement-celebration" aria-live="polite" aria-atomic="true">
      <span className="achievement-celebration__fireworks" aria-hidden>
        {Array.from({ length: 12 }, (_, index) => <i key={index} />)}
      </span>
      <button className={`achievement-toast achievement-toast--${achievementRarity(active.xp)}`} type="button" onClick={open}>
        <span className="achievement-medallion achievement-toast__medallion" aria-hidden><span>{active.icon}</span></span>
        <span className="achievement-toast__body">
          <span className="achievement-toast__eyebrow"><Trophy size={14} /> Новая ачивка</span>
          <strong>{active.title}</strong>
          <span>{active.text}</span>
        </span>
      </button>
    </div>
  );
}

function achievementRarity(xp: number) {
  if (xp >= 350) return 'platinum';
  if (xp >= 200) return 'gold';
  if (xp >= 100) return 'silver';
  return 'bronze';
}
