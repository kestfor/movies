import type { Comment } from '../types/api';

const BASE_INDENT_PX = 14;
const INDENT_RATIO = 0.72;

export function commentIndentForDepth(depth: number): number {
  if (depth <= 0) return 0;
  return Math.round(BASE_INDENT_PX * INDENT_RATIO ** (depth - 1));
}

export function countComments(comments: Comment[]): number {
  return comments.reduce(
    (total, comment) => total + 1 + countComments(comment.replies || []),
    0,
  );
}

export function commentCountLabel(count: number): string {
  const lastTwo = count % 100;
  const last = count % 10;
  const noun = lastTwo >= 11 && lastTwo <= 14
    ? 'комментариев'
    : last === 1
      ? 'комментарий'
      : last >= 2 && last <= 4
        ? 'комментария'
        : 'комментариев';
  return `${count} ${noun}`;
}
