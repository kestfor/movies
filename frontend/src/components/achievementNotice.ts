import type { UnseenAchievements } from '../types/api';

export type UnlockNotice = {
  ids: string[];
  awardID?: string;
  title: string;
  text: string;
  icon: string;
  xp: number;
};

export function nextUnlockNotice(
  unseen: UnseenAchievements,
  announcedIDs: ReadonlySet<string>,
): UnlockNotice | null {
  const achievement = unseen.items.find(
    (item) => item.award_id && !announcedIDs.has(item.award_id),
  );
  if (achievement?.award_id) {
    return {
      ids: [achievement.award_id],
      awardID: achievement.award_id,
      title: achievement.title || 'Секретное достижение',
      text: `Получено · +${achievement.xp || 0} XP`,
      icon: achievement.icon || '🏆',
      xp: achievement.xp || 0,
    };
  }

  const ids = (unseen.backfill_award_ids || []).filter((id) => !announcedIDs.has(id));
  if (!ids.length) return null;

  return {
    ids,
    title: 'Найдены достижения',
    text: `${ids.length} уже были вами получены`,
    icon: '🎉',
    xp: 200,
  };
}
