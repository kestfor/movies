import { Link } from 'react-router-dom';
import type { FeedItem, RatingWithUser, Title } from '../types/api';
import { ScorePill } from './Ui';

export const posterURL = (path?: string) => (path ? `https://image.tmdb.org/t/p/w342${path}` : '');

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

export function FeedCard({ item }: { item: FeedItem }) {
  return (
    <article className="feed-card">
      <div className="feed-card__user">
        <Avatar name={item.user.first_name} url={item.user.photo_url} />
        <span>{item.user.first_name}</span>
        <span className="muted">оценил</span>
        <ScorePill value={item.avg_score} />
      </div>
      <TitleRow title={item.title} />
      <ScoreDetails scores={item.scores} />
    </article>
  );
}

export function Avatar({ name, url }: { name: string; url?: string }) {
  return url ? <img className="avatar" src={url} alt="" /> : <span className="avatar">{name.slice(0, 1).toUpperCase()}</span>;
}

export function RatingCard({ rating }: { rating: RatingWithUser }) {
  return (
    <details className="rating-card">
      <summary>
        <span className="rating-card__user">
          <Avatar name={rating.user.first_name} url={rating.user.photo_url} />
          {rating.user.first_name}
        </span>
        <ScorePill value={rating.avg_score} />
      </summary>
      <ScoreDetails scores={rating.scores} />
    </details>
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
