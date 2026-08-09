export type ProfileThemeKey = 'viewer' | 'cinephile' | 'expert' | 'curator' | 'legend' | 'diamond';

type ProfileTheme = {
  key: ProfileThemeKey;
  step: 1 | 2 | 3;
};

const bands: Array<{ start: number; key: ProfileThemeKey }> = [
  { start: 15, key: 'diamond' },
  { start: 13, key: 'legend' },
  { start: 10, key: 'curator' },
  { start: 7, key: 'expert' },
  { start: 4, key: 'cinephile' },
  { start: 1, key: 'viewer' },
];

export function profileThemeForLevel(rawLevel: number): ProfileTheme {
  const level = Math.max(1, Math.floor(rawLevel || 1));
  const band = bands.find((candidate) => level >= candidate.start) || bands[bands.length - 1];
  const step = Math.min(3, level - band.start + 1) as 1 | 2 | 3;
  return { key: band.key, step };
}
