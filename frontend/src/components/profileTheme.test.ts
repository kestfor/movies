import { describe, expect, it } from 'vitest';
import { profileThemeForLevel } from './profileTheme';

describe('profileThemeForLevel', () => {
  it('maps level bands to their profile themes', () => {
    expect([1, 4, 7, 10, 13, 15].map((level) => profileThemeForLevel(level).key)).toEqual([
      'viewer',
      'cinephile',
      'expert',
      'curator',
      'legend',
      'diamond',
    ]);
  });

  it('increases visual intensity within a rank and caps it', () => {
    expect([7, 8, 9, 16, 20].map((level) => profileThemeForLevel(level).step)).toEqual([1, 2, 3, 2, 3]);
  });
});
