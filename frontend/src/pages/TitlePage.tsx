import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Bookmark, BookmarkCheck, Share2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { api } from '../api/client';
import { Comments } from '../components/Comments';
import { RatingEditor } from '../components/RatingEditor';
import { Accordion, Poster, RatingCard, ScoreDetails } from '../components/TitleBits';
import { Button, EmptyState, ErrorState, LoadingState, ScorePill } from '../components/Ui';
import { useToast } from '../components/Toast';
import { getActiveTitleTransitionByRef, getActiveTitleTransitionNameByRef } from '../lib/transitions';
import type { TitleTransitionSnapshot } from '../lib/transitions';
import { haptic, shareTitle } from '../lib/telegram';
import { useWatchlistMutation } from '../hooks/useWatchlistMutation';

export function TitlePage() {
  const { type = '', id = '' } = useParams();
  const [searchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { showToast, showError } = useToast();
  const watchlistMutation = useWatchlistMutation();
  const [ratingEditorOpen, setRatingEditorOpen] = useState<boolean | undefined>(undefined);
  const [ratingCelebrating, setRatingCelebrating] = useState(false);
  const celebrationTimer = useRef<number | null>(null);
  const titleKey = ['title', type, id];
  const commentsKey = ['comments', type, id];
  const card = useQuery({ queryKey: titleKey, queryFn: () => api.title(type, id) });
  const comments = useQuery({ queryKey: commentsKey, queryFn: () => api.comments(type, id) });
  const criteria = useQuery({ queryKey: ['criteria'], queryFn: api.criteria });
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });
  const sharedTransitionName = getActiveTitleTransitionNameByRef(type, id);
  const activeTransition = getActiveTitleTransitionByRef(type, id);

  useEffect(() => () => {
    if (celebrationTimer.current !== null) window.clearTimeout(celebrationTimer.current);
  }, []);

  const refreshTitle = () => {
    queryClient.invalidateQueries({ queryKey: titleKey });
    queryClient.invalidateQueries({ queryKey: ['feed'] });
    queryClient.invalidateQueries({ queryKey: ['profile'] });
    queryClient.invalidateQueries({ queryKey: ['profileWatchlist'] });
    queryClient.invalidateQueries({ queryKey: ['discover'] });
    queryClient.invalidateQueries({ queryKey: ['recommendations'] });
  };

  const saveRating = useMutation({
    mutationFn: (scores: Record<string, number>) => api.putRating(type, id, scores),
    onSuccess: () => {
      haptic('success');
      showToast('Оценка сохранена');
      setRatingEditorOpen(true);
      setRatingCelebrating(true);
      queryClient.setQueryData(titleKey, (old: typeof card.data) => old ? { ...old, in_watchlist: false } : old);
      refreshTitle();
      if (celebrationTimer.current !== null) window.clearTimeout(celebrationTimer.current);
      const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      celebrationTimer.current = window.setTimeout(() => {
        setRatingCelebrating(false);
        setRatingEditorOpen(false);
        celebrationTimer.current = null;
      }, reducedMotion ? 280 : 650);
    },
    onError: (error) => {
      setRatingCelebrating(false);
      haptic('error');
      showError(error, 'Оценку не удалось сохранить');
    },
  });
  const deleteRating = useMutation({
    mutationFn: () => api.deleteRating(type, id),
    onSuccess: () => {
      if (celebrationTimer.current !== null) {
        window.clearTimeout(celebrationTimer.current);
        celebrationTimer.current = null;
      }
      setRatingCelebrating(false);
      haptic('success');
      showToast('Оценка удалена', 'warning');
      setRatingEditorOpen(true);
      refreshTitle();
    },
    onError: (error) => {
      haptic('error');
      showError(error, 'Оценку не удалось удалить');
    },
  });
  const createComment = useMutation({
    mutationFn: ({ body, parentID }: { body: string; parentID?: number }) => api.postComment(type, id, body, parentID),
    onSuccess: () => {
      haptic('success');
      showToast('Комментарий отправлен');
      queryClient.invalidateQueries({ queryKey: commentsKey });
      queryClient.invalidateQueries({ queryKey: titleKey });
    },
    onError: (error) => {
      haptic('error');
      showError(error, 'Комментарий не отправлен');
    },
  });
  const updateComment = useMutation({
    mutationFn: ({ commentID, body }: { commentID: number; body: string }) => api.patchComment(commentID, body),
    onSuccess: () => {
      haptic('success');
      showToast('Комментарий обновлён');
      queryClient.invalidateQueries({ queryKey: commentsKey });
    },
    onError: (error) => {
      haptic('error');
      showError(error, 'Комментарий не обновлён');
    },
  });
  const removeComment = useMutation({
    mutationFn: (commentID: number) => api.deleteComment(commentID),
    onSuccess: () => {
      haptic('success');
      showToast('Комментарий удалён', 'warning');
      queryClient.invalidateQueries({ queryKey: commentsKey });
    },
    onError: (error) => {
      haptic('error');
      showError(error, 'Комментарий не удалён');
    },
  });

  if (card.isLoading || criteria.isLoading) {
    if (sharedTransitionName && activeTransition?.title) {
      return <TitleHeroPreview title={activeTransition.title} sharedTransitionName={sharedTransitionName} />;
    }
    return <LoadingState label="Открываем карточку" />;
  }
  if (card.isError) return <ErrorState error={card.error} />;
  if (criteria.isError) return <ErrorState error={criteria.error} />;
  if (!card.data) return <EmptyState title="Тайтл не найден" />;
  if (!criteria.data) return <EmptyState title="Критерии не загружены" />;

  const title = card.data.title;
  const labels = Object.fromEntries(criteria.data.criteria.map((criterion) => [criterion.code, criterion.name]));
  const editorOpen = ratingEditorOpen ?? !card.data.my_rating;
  const sharedTransitionStyle = sharedTransitionName
    ? ({ viewTransitionName: sharedTransitionName } as CSSProperties)
    : undefined;

  return (
    <>
      <section className={`title-hero ${sharedTransitionName ? 'title-hero--shared' : ''}`}>
        <div className="title-hero__top" style={sharedTransitionStyle}>
          <Poster title={title} />
          <div className="title-hero__name">
            <div className="title-hero__eyebrow">
              <span className="muted">{title.media_type === 'tv' ? 'Сериал' : 'Фильм'}</span>
              <button
                className="icon-button title-share-button"
                type="button"
                onClick={() => shareTitle(title.media_type, title.tmdb_id, title.title)}
                aria-label={`Поделиться: ${title.title}`}
                title="Поделиться"
              >
                <Share2 size={16} aria-hidden />
              </button>
            </div>
            <h2>{title.title}</h2>
            {title.release_year ? <p className="muted">{title.release_year}</p> : null}
            <div className="title-hero__scores">
              <div>
                <span className="muted">Моя</span>
                <ScorePill value={card.data.my_rating?.avg_score} />
              </div>
              <div>
                <span className="muted">Друзья</span>
                <ScorePill value={card.data.friends_avg?.overall} muted={!card.data.friends_avg} />
              </div>
            </div>
          </div>
        </div>
        <div className="title-hero__details">
          {title.genres?.length ? <p className="title-hero__genres muted">{title.genres.join(', ')}</p> : null}
          {title.overview ? <p className="title-hero__overview">{title.overview}</p> : null}
        </div>
      </section>
      {!card.data.my_rating ? (
        <div className="title-watchlist-action">
          <Button
            variant="ghost"
            disabled={watchlistMutation.isPending}
            onClick={() => watchlistMutation.mutate({ title, inWatchlist: !card.data.in_watchlist })}
          >
            {card.data.in_watchlist ? <BookmarkCheck size={18} aria-hidden /> : <Bookmark size={18} aria-hidden />}
            {card.data.in_watchlist ? 'В списке «Хочу посмотреть»' : 'Хочу посмотреть'}
          </Button>
        </div>
      ) : null}
      <RatingEditor
        criteria={criteria.data.criteria}
        rating={card.data.my_rating}
        saving={saveRating.isPending}
        celebrating={ratingCelebrating}
        open={editorOpen}
        onOpenChange={setRatingEditorOpen}
        onSave={(scores) => saveRating.mutate(scores)}
        onDelete={() => deleteRating.mutate()}
      />
      {card.data.friends_avg ? (
        <section className="panel">
          <Accordion
            title="Средняя друзей"
            summary="По критериям"
            action={<ScorePill value={card.data.friends_avg.overall} />}
            defaultOpen={false}
          >
            <ScoreDetails scores={card.data.friends_avg.by_criteria} labels={labels} />
          </Accordion>
        </section>
      ) : null}
      <section className="panel">
        {card.data.friends_ratings.length ? (
          <Accordion title="Оценки друзей" defaultOpen={false}>
            <div className="stack tight">
              {card.data.friends_ratings.map((rating) => (
                <RatingCard key={rating.user.uuid} rating={rating} labels={labels} />
              ))}
            </div>
          </Accordion>
        ) : (
          <>
            <div className="panel__header">
              <div>
                <h2>Оценки друзей</h2>
              </div>
            </div>
            <p className="muted">Оценки появятся после добавления друзей.</p>
          </>
        )}
      </section>
      {comments.isLoading ? <LoadingState label="Загружаем комментарии" /> : null}
      {comments.isError ? <ErrorState error={comments.error} /> : null}
      {comments.data ? (
        <Comments
          comments={comments.data.comments}
          activeCommentID={Number(searchParams.get('comment_id')) || undefined}
          me={me.data}
          posting={createComment.isPending}
          onCreate={(body, parentID) => createComment.mutate({ body, parentID })}
          onUpdate={(commentID, body) => updateComment.mutate({ commentID, body })}
          onDelete={(commentID) => removeComment.mutate(commentID)}
        />
      ) : null}
    </>
  );
}

function TitleHeroPreview({
  title,
  sharedTransitionName,
}: {
  title: TitleTransitionSnapshot;
  sharedTransitionName: string;
}) {
  return (
    <section
      className="title-hero title-hero--shared title-hero--preview"
    >
      <div className="title-hero__top" style={{ viewTransitionName: sharedTransitionName } as CSSProperties}>
        <Poster title={title} />
        <div className="title-hero__name">
          <span className="muted">{title.media_type === 'tv' ? 'Сериал' : 'Фильм'}</span>
          <h2>{title.title}</h2>
          {title.release_year ? <p className="muted">{title.release_year}</p> : null}
          <div className="title-hero__scores">
            <div className="skeleton-box" />
            <div className="skeleton-box" />
          </div>
        </div>
      </div>
      <div className="title-hero__details">
        {title.genres?.length ? <p className="title-hero__genres muted">{title.genres.join(', ')}</p> : null}
        {title.overview ? <p className="title-hero__overview">{title.overview}</p> : <div className="skeleton-line" />}
      </div>
    </section>
  );
}
