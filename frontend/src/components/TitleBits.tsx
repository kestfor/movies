import { ChevronDown } from 'lucide-react';
import { useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import type { MouseEvent, ReactNode } from 'react';
import { flushSync } from 'react-dom';
import { Link, useNavigate } from 'react-router-dom';
import {
  setActiveTitleTransition,
  startViewTransition,
  suppressNextPageTransition,
  titleRef,
  titleTransitionName,
} from '../lib/transitions';
import type { FeedItem, RatingWithUser, Title, User } from '../types/api';
import { ScorePill } from './Ui';
import { api } from '../api/client';
import { haptic } from '../lib/telegram';
import { shouldConfirmTitleLoaded } from '../lib/titleOpenFeedback';

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
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [opening, setOpening] = useState(false);
  const to = `/title/${title.media_type}/${title.tmdb_id}`;

  const openTitle = async (event: MouseEvent<HTMLAnchorElement>) => {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) return;

    event.preventDefault();
    if (opening) return;

    const target = event.currentTarget;
    const titleKey = ['title', title.media_type, String(title.tmdb_id)];
    const hadCachedTitle = queryClient.getQueryData(titleKey) !== undefined;
    setActiveTitleTransition(title);
    setOpening(true);
    haptic('medium');

    try {
      await queryClient.ensureQueryData({
        queryKey: titleKey,
        queryFn: () => api.title(title.media_type, title.tmdb_id),
      });
    } catch {
      setOpening(false);
      haptic('warning');
      suppressNextPageTransition();
      navigate(to, { state: { pageDirection: 'forward' } });
      return;
    }

    if (shouldConfirmTitleLoaded(hadCachedTitle)) haptic('light');
    target.style.setProperty('view-transition-name', titleTransitionName(title));

    const transition = startViewTransition(() => {
      suppressNextPageTransition();
      flushSync(() => navigate(to, { state: { pageDirection: 'forward' } }));
    });
    if (!transition) {
      target.style.removeProperty('view-transition-name');
      setOpening(false);
      return;
    }
    transition.finished.finally(() => {
      target.style.removeProperty('view-transition-name');
      setOpening(false);
    });
  };

  return (
    <Link
      className={`title-row ${opening ? 'title-row--opening' : ''}`}
      to={to}
      data-title-transition-id={titleRef(title.media_type, title.tmdb_id)}
      onClick={openTitle}
      aria-busy={opening}
    >
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
          <div className="feed-card__event">
            <span>поставил(а) оценку</span>
            <span className="rating-date">{formatRatingDate(item.created_at, item.updated_at)}</span>
          </div>
        </div>
        <ScorePill value={item.avg_score} />
      </div>
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
      <span className="user-link__name">{user.first_name}</span>
      {!compact && user.username ? <span className="user-link__username muted">@{user.username}</span> : null}
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
  const [open, setOpen] = useState(false);

  return (
    <article className={`rating-card ${open ? 'rating-card--open' : ''}`}>
      <button
        className="rating-card__head"
        type="button"
        onClick={() => {
          haptic('light');
          setOpen((value) => !value);
        }}
        aria-label="Показать детали оценки"
        aria-expanded={open}
      >
        <div className="rating-card__meta">
          <span className="rating-card__user">
            <UserLink user={rating.user} compact />
          </span>
          <span className="rating-date">{formatRatingDate(rating.created_at, rating.updated_at)}</span>
        </div>
        <div className="rating-card__actions">
          <ScorePill value={rating.avg_score} />
          <span className="rating-card__toggle" aria-hidden="true">
            <ChevronDown size={17} aria-hidden="true" />
          </span>
        </div>
      </button>
      <div className="rating-card__details">
        <div className="rating-card__details-inner">
          <ScoreDetails scores={rating.scores} labels={labels} />
        </div>
      </div>
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
  open: controlledOpen,
  onOpenChange,
  action,
  className = '',
}: {
  title: string;
  summary?: string;
  children: ReactNode;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  action?: ReactNode;
  className?: string;
}) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(defaultOpen);
  const open = controlledOpen ?? uncontrolledOpen;
  const toggle = () => {
    const next = !open;
    haptic('light');
    if (onOpenChange) onOpenChange(next);
    else setUncontrolledOpen(next);
  };

  return (
    <div className={`accordion ${open ? 'accordion--open' : ''} ${className}`}>
      <button
        className="accordion__trigger"
        type="button"
        onClick={toggle}
        aria-expanded={open}
      >
        <span>{title}</span>
        {summary ? <span className="muted">{summary}</span> : null}
        {action ? <span className="accordion__action">{action}</span> : null}
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
