import { describe, expect, it } from 'vitest';
import { updateWatchlistData } from './watchlistCache';
import type { Title } from '../types/api';

const title: Title = {
  tmdb_id: 603,
  media_type: 'movie',
  title: 'Матрица',
};

describe('updateWatchlistData', () => {
  it('updates the same title across catalog and title-card shapes', () => {
    const data = {
      pages: [{ items: [{ title, in_watchlist: false }] }],
      title,
      in_watchlist: false,
    };

    expect(updateWatchlistData(data, title, true)).toEqual({
      pages: [{ items: [{ title, in_watchlist: true }] }],
      title,
      in_watchlist: true,
    });
  });

  it('removes an owner watchlist row optimistically', () => {
    const data = {
      pages: [{ items: [{ title, added_at: '2026-08-02T10:00:00Z' }] }],
    };

    expect(updateWatchlistData(data, title, false)).toEqual({
      pages: [{ items: [] }],
    });
  });

  it('does not change another title', () => {
    const another: Title = { tmdb_id: 1396, media_type: 'tv', title: 'Во все тяжкие' };
    const data = { items: [{ title: another, in_watchlist: false }] };
    expect(updateWatchlistData(data, title, true)).toEqual(data);
  });
});
