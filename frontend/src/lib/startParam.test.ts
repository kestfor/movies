import { describe, expect, it } from 'vitest';
import { titlePathFromStartParam, titleStartParam, titleTelegramDeepLink } from './startParam';

describe('titlePathFromStartParam', () => {
  it('builds a Telegram Mini App deep link', () => {
    expect(titleStartParam('movie', 313369)).toBe('title_movie_313369');
    expect(titleTelegramDeepLink('moviesclubtechbot', 'moviesclub', 'movie', 313369)).toBe(
      'https://t.me/moviesclubtechbot/moviesclub?startapp=title_movie_313369',
    );
  });

  it('builds movie and TV title paths', () => {
    expect(titlePathFromStartParam('title_movie_313369')).toBe('/title/movie/313369');
    expect(titlePathFromStartParam('title_tv_1396')).toBe('/title/tv/1396');
  });

  it('rejects unsupported or malformed parameters', () => {
    expect(titlePathFromStartParam('uid_00000000-0000-0000-0000-000000000000')).toBeNull();
    expect(titlePathFromStartParam('title_person_1')).toBeNull();
    expect(titlePathFromStartParam('title_movie_0')).toBeNull();
    expect(titlePathFromStartParam('title_movie_313369_extra')).toBeNull();
  });
});
