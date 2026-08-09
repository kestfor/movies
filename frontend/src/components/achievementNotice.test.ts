import { describe, expect, it } from 'vitest';
import type { UnseenAchievements } from '../types/api';
import { nextUnlockNotice } from './achievementNotice';

const unseen: UnseenAchievements = {
  items: [
    { award_id: 'award-1', title: 'Первая', icon: '🍿', xp: 50, secret: false, unlocked: true, sort_order: 1 },
    { award_id: 'award-2', title: 'Вторая', icon: '🎬', xp: 100, secret: false, unlocked: true, sort_order: 2 },
  ],
  backfill_count: 2,
  backfill_award_ids: ['backfill-1', 'backfill-2'],
};

describe('nextUnlockNotice', () => {
  it('does not select an achievement already announced in this session', () => {
    expect(nextUnlockNotice(unseen, new Set(['award-1']))?.awardID).toBe('award-2');
  });

  it('does not repeat stale unseen data after all awards were announced', () => {
    const announced = new Set(['award-1', 'award-2', 'backfill-1', 'backfill-2']);

    expect(nextUnlockNotice(unseen, announced)).toBeNull();
  });

  it('groups only unannounced backfill awards', () => {
    const announced = new Set(['award-1', 'award-2', 'backfill-1']);

    expect(nextUnlockNotice(unseen, announced)?.ids).toEqual(['backfill-2']);
  });
});
