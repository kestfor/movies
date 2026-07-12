import { ChevronDown } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import type { FeedItem, RatingWithUser, Title, User } from '../types/api';
import { ScorePill } from './Ui';

export const posterURL = (path?: string) => path || '';

export function Poster({ title }: { title: Title }) {
  const url = posterURL(title.poster_path);
  return url ? (
    <img className="poster" src={url} alt="" loading="lazy" />
  ) : (
    <div className="poster poster--empty">{title.media_type === 'tv' ? 'TV' : 'Кино'}</div>
  );
}

export function TitleRow({ title, score }: { title: Title; score?: number }) {
  return (
    <Link className="title-row" to={`/title/${title.media_type}/${title.tmdb_id}`}>
      <Poster title={title} />
      <div className="title-row__body">
        <div className="title-row__title">{title.title}</div>
        <div className="muted">
          {title.media_type === 'tv' ? 'Сериал' : 'Фильм'}
          {title.release_year ? ` · ${title.release_year}` : ''}
        </div>
        {title.overview ? <p>{title.overview}</p> : null}
      </div>
      {score !== undefined ? <ScorePill value={score} /> : null}
    </Link>
  );
}

export function FeedCard({ item, labels }: { item: FeedItem; labels?: Record<string, string> }) {
  return (
    <article className="feed-card">
      <div className="feed-card__user">
        <div className="feed-card__actor">
          <UserLink user={item.user} />
          <span className="muted">поставил(а) оценку</span>
        </div>
        <ScorePill value={item.avg_score} />
      </div>
      <div className="rating-date">{formatRatingDate(item.created_at, item.updated_at)}</div>
      <TitleRow title={item.title} />
      <Accordion title="Детали оценки" summary={`${Object.keys(item.scores).length} оценок`}>
        <ScoreDetails scores={item.scores} labels={labels} />
      </Accordion>
    </article>
  );
}

export function Avatar({ name, url }: { name: string; url?: string }) {
  return url ? <img className="avatar" src={url} alt="" /> : <span className="avatar">{name.slice(0, 1).toUpperCase()}</span>;
}

export function UserLink({ user, compact = false }: { user: User; compact?: boolean }) {
  const content = (
    <>
      <Avatar name={user.first_name} url={user.photo_url} />
      <span>{user.first_name}</span>
      {!compact && user.username ? <span className="muted">@{user.username}</span> : null}
    </>
  );

  if (!user.uuid) {
    return <span className="user-link">{content}</span>;
  }
  return (
    <Link className="user-link" to={`/profile/${user.uuid}`} onClick={(event) => event.stopPropagation()}>
      {content}
    </Link>
  );
}

export function RatingCard({ rating, labels }: { rating: RatingWithUser; labels?: Record<string, string> }) {
  return (
    <article className="rating-card">
      <div className="rating-card__head">
        <div className="rating-card__meta">
          <span className="rating-card__user">
            <UserLink user={rating.user} compact />
          </span>
          <span className="rating-date">{formatRatingDate(rating.created_at, rating.updated_at)}</span>
        </div>
        <ScorePill value={rating.avg_score} />
      </div>
      <Accordion title="Разбивка оценки" summary={`${Object.keys(rating.scores).length} оценок`}>
        <ScoreDetails scores={rating.scores} labels={labels} />
      </Accordion>
    </article>
  );
}

export function ScoreDetails({ scores, labels }: { scores: Record<string, number>; labels?: Record<string, string> }) {
  const entries = Object.entries(scores);
  if (!entries.length) return null;
  return (
    <div className="score-grid">
      {entries.map(([code, value]) => (
        <div key={code} className="score-grid__item">
          <span>{labels?.[code] || code}</span>
          <b>{value}</b>
        </div>
      ))}
    </div>
  );
}

export function Accordion({
  title,
  summary,
  children,
  defaultOpen = false,
  className = '',
}: {
  title: string;
  summary?: string;
  children: ReactNode;
  defaultOpen?: boolean;
  className?: string;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className={`accordion ${open ? 'accordion--open' : ''} ${className}`}>
      <button className="accordion__trigger" type="button" onClick={() => setOpen((value) => !value)} aria-expanded={open}>
        <span>{title}</span>
        {summary ? <span className="muted">{summary}</span> : null}
        <ChevronDown className="accordion__icon" size={17} aria-hidden="true" />
      </button>
      <div className="accordion__content">
        <div className="accordion__inner">{children}</div>
      </div>
    </div>
  );
}

export function formatRatingDate(createdAt: string, updatedAt?: string) {
  const source = updatedAt || createdAt;
  if (!source) return '';

  const date = new Date(source);
  if (Number.isNaN(date.getTime())) return '';

  const now = new Date();
  const includeYear = date.getFullYear() !== now.getFullYear();
  const formatted = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    ...(includeYear ? { year: 'numeric' } : {}),
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);

  return formatted.replace(',', '');
}
