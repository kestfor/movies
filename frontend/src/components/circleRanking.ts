import type { AchievementLeaderboard } from '../types/api';

type RankingEntry = AchievementLeaderboard['items'][number];

export function rankingPreview(items: RankingEntry[], targetUUID: string): RankingEntry[] {
  const preview = items.slice(0, 3);
  const target = items.find((item) => item.user.uuid === targetUUID);
  if (target && !preview.some((item) => item.user.uuid === targetUUID)) preview.push(target);
  return preview;
}

export function nextRankGap(items: RankingEntry[], targetUUID: string) {
  const target = items.find((item) => item.user.uuid === targetUUID);
  if (!target || target.rank <= 1) return null;

  let ahead: RankingEntry | undefined;
  for (const item of items) {
    if (item.rank < target.rank && (!ahead || item.rank > ahead.rank)) ahead = item;
  }
  if (!ahead) return null;
  return {
    rank: ahead.rank,
    xp: Math.max(0, ahead.total_xp - target.total_xp),
  };
}
