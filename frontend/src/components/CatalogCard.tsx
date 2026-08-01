import { Bookmark, BookmarkCheck } from 'lucide-react';
import type { CatalogItem } from '../types/api';
import { TitleRow } from './TitleBits';

export function CatalogCard({
  item,
  writable = true,
  pending = false,
  onToggle,
}: {
  item: CatalogItem;
  writable?: boolean;
  pending?: boolean;
  onToggle?: (next: boolean) => void;
}) {
  const active = item.in_watchlist;
  return (
    <article className="catalog-card">
      <TitleRow title={item.title} />
      {item.reason ? <p className="catalog-card__reason">{item.reason}</p> : null}
      {writable ? (
        <button
          className={`catalog-card__bookmark ${active ? 'is-active' : ''}`}
          type="button"
          disabled={pending}
          aria-label={active ? 'Удалить из «Хочу посмотреть»' : 'Добавить в «Хочу посмотреть»'}
          aria-pressed={active}
          onClick={() => onToggle?.(!active)}
        >
          {active ? <BookmarkCheck size={20} aria-hidden /> : <Bookmark size={20} aria-hidden />}
        </button>
      ) : null}
    </article>
  );
}
