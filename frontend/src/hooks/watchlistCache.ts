import type { Title } from '../types/api';

export function updateWatchlistData(value: unknown, title: Title, inWatchlist: boolean): unknown {
  if (Array.isArray(value)) {
    return value
      .filter((item) => !(!inWatchlist && isWatchlistRow(item) && matchesTitle(item.title, title)))
      .map((item) => updateWatchlistData(item, title, inWatchlist));
  }
  if (!value || typeof value !== 'object') return value;
  const record = value as Record<string, unknown>;
  const next: Record<string, unknown> = {};
  Object.entries(record).forEach(([key, child]) => {
    next[key] = updateWatchlistData(child, title, inWatchlist);
  });
  if (record.title && typeof record.title === 'object' && matchesTitle(record.title, title)) {
    next.in_watchlist = inWatchlist;
  }
  return next;
}

function isWatchlistRow(value: unknown): value is { title: unknown; added_at: string } {
  return Boolean(value && typeof value === 'object' && 'title' in value && 'added_at' in value);
}

function matchesTitle(value: unknown, title: Title) {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as Partial<Title>;
  return candidate.media_type === title.media_type && candidate.tmdb_id === title.tmdb_id;
}
