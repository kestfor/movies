import { describe, expect, it } from 'vitest';
import { availableFriendSelection, normalizeFriendSelection, sameFriendSelection, watchTogetherPeopleLabel } from './watchTogether';

describe('watch together helpers', () => {
  it('normalizes and filters friend selections', () => {
    expect(normalizeFriendSelection(['b', 'a', 'b', ''])).toEqual(['a', 'b']);
    expect(availableFriendSelection(['b', 'a'], new Set(['a']))).toEqual(['a']);
    expect(sameFriendSelection(['a', 'b'], ['a', 'b'])).toBe(true);
    expect(sameFriendSelection(['a', 'b'], ['b', 'a'])).toBe(false);
  });

  it('builds a compact participant label', () => {
    const users = [
      { uuid: 'me', first_name: 'Илья' },
      { uuid: 'a', first_name: 'Аня' },
      { uuid: 'm', first_name: 'Миша' },
      { uuid: 'o', first_name: 'Оля' },
    ];
    expect(watchTogetherPeopleLabel(users)).toBe('Хотят посмотреть: вы, Аня, Миша и ещё 1');
    expect(watchTogetherPeopleLabel([])).toBe('Хотят посмотреть');
  });
});
