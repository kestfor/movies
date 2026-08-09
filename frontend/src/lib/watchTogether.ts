import type { User } from '../types/api';

export function normalizeFriendSelection(values: readonly string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).sort();
}

export function availableFriendSelection(values: readonly string[], availableUUIDs: ReadonlySet<string>) {
  return normalizeFriendSelection(values).filter((value) => availableUUIDs.has(value));
}

export function sameFriendSelection(left: readonly string[], right: readonly string[]) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

export function watchTogetherPeopleLabel(users: readonly User[], visibleCount = 3) {
  if (!users.length) return 'Хотят посмотреть';
  const names = users.map((user, index) => index === 0 ? 'вы' : user.first_name);
  const visible = names.slice(0, visibleCount);
  const remaining = names.length - visible.length;
  return `Хотят посмотреть: ${visible.join(', ')}${remaining > 0 ? ` и ещё ${remaining}` : ''}`;
}
