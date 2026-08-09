import { describe, expect, it } from 'vitest';
import type { AchievementLeaderboard } from '../types/api';
import { nextRankGap, rankingPreview } from './circleRanking';

const items: AchievementLeaderboard['items'] = [
  { rank: 1, user: { uuid: 'anna', first_name: 'Анна' }, total_xp: 2400, level: 7, unlocked_count: 20 },
  { rank: 2, user: { uuid: 'max', first_name: 'Макс' }, total_xp: 2100, level: 7, unlocked_count: 18 },
  { rank: 3, user: { uuid: 'olga', first_name: 'Ольга' }, total_xp: 1900, level: 6, unlocked_count: 16 },
  { rank: 4, user: { uuid: 'ivan', first_name: 'Иван' }, total_xp: 1700, level: 6, unlocked_count: 14 },
];

describe('circle ranking helpers', () => {
  it('keeps the opened profile visible outside the top three', () => {
    expect(rankingPreview(items, 'ivan').map((item) => item.user.uuid)).toEqual(['anna', 'max', 'olga', 'ivan']);
  });

  it('calculates the XP gap to the next rank', () => {
    expect(nextRankGap(items, 'ivan')).toEqual({ rank: 3, xp: 200 });
  });

  it('has no next-rank gap for the leader', () => {
    expect(nextRankGap(items, 'anna')).toBeNull();
  });
});
