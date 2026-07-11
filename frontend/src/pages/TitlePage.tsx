import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { api } from '../api/client';
import { Comments } from '../components/Comments';
import { RatingEditor } from '../components/RatingEditor';
import { Poster, RatingCard, ScoreDetails } from '../components/TitleBits';
import { EmptyState, ErrorState, LoadingState, PageHeader, ScorePill } from '../components/Ui';
import { haptic } from '../lib/telegram';

export function TitlePage() {
  const { type = '', id = '' } = useParams();
  const queryClient = useQueryClient();
  const titleKey = ['title', type, id];
  const commentsKey = ['comments', type, id];
  const card = useQuery({ queryKey: titleKey, queryFn: () => api.title(type, id) });
  const comments = useQuery({ queryKey: commentsKey, queryFn: () => api.comments(type, id) });
  const criteria = useQuery({ queryKey: ['criteria'], queryFn: api.criteria });
  const me = useQuery({ queryKey: ['me'], queryFn: api.me });

  const refreshTitle = () => {
    queryClient.invalidateQueries({ queryKey: titleKey });
    queryClient.invalidateQueries({ queryKey: ['feed'] });
    queryClient.invalidateQueries({ queryKey: ['profile'] });
  };

  const saveRating = useMutation({
    mutationFn: (scores: Record<string, number>) => api.putRating(type, id, scores),
    onSuccess: () => {
      haptic('success');
      refreshTitle();
    },
  });
  const deleteRating = useMutation({
    mutationFn: () => api.deleteRating(type, id),
    onSuccess: refreshTitle,
  });
  const createComment = useMutation({
    mutationFn: ({ body, parentID }: { body: string; parentID?: number }) => api.postComment(type, id, body, parentID),
    onSuccess: () => {
      haptic('success');
      queryClient.invalidateQueries({ queryKey: commentsKey });
      queryClient.invalidateQueries({ queryKey: titleKey });
    },
  });
  const updateComment = useMutation({
    mutationFn: ({ commentID, body }: { commentID: number; body: string }) => api.patchComment(commentID, body),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: commentsKey }),
  });
  const removeComment = useMutation({
    mutationFn: (commentID: number) => api.deleteComment(commentID),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: commentsKey }),
  });

  if (card.isLoading || criteria.isLoading) return <LoadingState label="Открываем карточку" />;
  if (card.isError) return <ErrorState error={card.error} />;
  if (criteria.isError) return <ErrorState error={criteria.error} />;
  if (!card.data) return <EmptyState title="Тайтл не найден" />;
  if (!criteria.data) return <EmptyState title="Критерии не загружены" />;

  const title = card.data.title;
  const labels = Object.fromEntries(criteria.data.criteria.map((criterion) => [criterion.code, criterion.name]));

  return (
    <>
      <PageHeader title={title.title} subtitle={`${title.media_type === 'tv' ? 'Сериал' : 'Фильм'}${title.release_year ? ` · ${title.release_year}` : ''}`} />
      <section className="title-hero">
        <Poster title={title} />
        <div>
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
          {title.genres?.length ? <p className="muted">{title.genres.join(', ')}</p> : null}
          {title.overview ? <p>{title.overview}</p> : null}
        </div>
      </section>
      {card.data.friends_avg ? (
        <section className="panel">
          <div className="panel__header">
            <div>
              <h2>Средняя друзей</h2>
              <p>По заполненным критериям</p>
            </div>
            <ScorePill value={card.data.friends_avg.overall} />
          </div>
          <ScoreDetails scores={card.data.friends_avg.by_criteria} labels={labels} />
        </section>
      ) : null}
      <RatingEditor
        criteria={criteria.data.criteria}
        rating={card.data.my_rating}
        saving={saveRating.isPending}
        onSave={(scores) => saveRating.mutate(scores)}
        onDelete={() => deleteRating.mutate()}
      />
      <section className="panel">
        <div className="panel__header">
          <div>
            <h2>Оценки друзей</h2>
            <p>{card.data.friends_ratings.length || 'Пока нет'}</p>
          </div>
        </div>
        {card.data.friends_ratings.length ? (
          <div className="stack tight">
            {card.data.friends_ratings.map((rating) => (
              <RatingCard key={rating.user.id} rating={rating} />
            ))}
          </div>
        ) : (
          <p className="muted">Оценки появятся после добавления друзей.</p>
        )}
      </section>
      {comments.isLoading ? <LoadingState label="Загружаем комментарии" /> : null}
      {comments.isError ? <ErrorState error={comments.error} /> : null}
      {comments.data ? (
        <Comments
          comments={comments.data.comments}
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
