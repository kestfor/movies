import { describe, expect, it } from 'vitest';
import type { Comment } from '../types/api';
import { commentCountLabel, commentIndentForDepth, countComments } from './commentThread';

const comment = (id: number, replies: Comment[] = []): Comment => ({
  id,
  title_id: 1,
  user: { uuid: `user-${id}`, first_name: `User ${id}` },
  body: `Comment ${id}`,
  is_deleted: false,
  created_at: '2026-08-09T10:00:00Z',
  updated_at: '2026-08-09T10:00:00Z',
  replies,
});

describe('commentIndentForDepth', () => {
  it('reduces each nested offset without consuming unbounded width', () => {
    expect([1, 2, 3, 4, 5, 10, 12].map(commentIndentForDepth)).toEqual([14, 10, 7, 5, 4, 1, 0]);
  });
});

describe('comment thread summary', () => {
  it('counts every nested comment', () => {
    expect(countComments([comment(1, [comment(2, [comment(3)]), comment(4)])])).toBe(4);
  });

  it('uses the correct Russian plural form', () => {
    expect([1, 2, 5, 11, 21].map(commentCountLabel)).toEqual([
      '1 комментарий',
      '2 комментария',
      '5 комментариев',
      '11 комментариев',
      '21 комментарий',
    ]);
  });
});
