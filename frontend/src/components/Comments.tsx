import { Check, Loader2, MessageCircle, Pencil, Reply, Send, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import type { Comment, User } from '../types/api';
import { UserLink } from './TitleBits';
import { haptic } from '../lib/telegram';
import { commentCountLabel, commentIndentForDepth, countComments } from './commentThread';

export function Comments({
  comments,
  me,
  posting,
  activeCommentID,
  onCreate,
  onUpdate,
  onDelete,
}: {
  comments: Comment[];
  me?: User;
  posting?: boolean;
  activeCommentID?: number;
  onCreate: (body: string, parentID?: number) => void;
  onUpdate: (id: number, body: string) => void;
  onDelete: (id: number) => void;
}) {
  const [body, setBody] = useState('');
  const commentCount = countComments(comments);

  return (
    <section className="panel comments-panel">
      <div className="panel__header">
        <div>
          <h2>Комментарии</h2>
          <p>{commentCount ? commentCountLabel(commentCount) : 'Начните обсуждение'}</p>
        </div>
        <span className="comments-panel__icon" aria-hidden="true"><MessageCircle size={19} /></span>
      </div>
      <CommentComposer
        value={body}
        onChange={setBody}
        onSubmit={() => {
          onCreate(body);
          setBody('');
        }}
        placeholder="Написать комментарий"
        submitLabel={posting ? 'Публикуем комментарий' : 'Отправить комментарий'}
        pending={posting}
        root
      />
      <div className="comments-list">
        {comments.map((comment) => (
          <CommentNode
            key={comment.id}
            comment={comment}
            activeCommentID={activeCommentID}
            me={me}
            onCreate={onCreate}
            onUpdate={onUpdate}
            onDelete={onDelete}
            depth={0}
          />
        ))}
      </div>
    </section>
  );
}

function CommentNode({
  comment,
  me,
  onCreate,
  onUpdate,
  onDelete,
  activeCommentID,
  depth,
}: {
  comment: Comment;
  me?: User;
  activeCommentID?: number;
  depth: number;
  onCreate: (body: string, parentID?: number) => void;
  onUpdate: (id: number, body: string) => void;
  onDelete: (id: number) => void;
}) {
  const [replying, setReplying] = useState(false);
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(comment.body);
  const ref = useRef<HTMLElement | null>(null);
  const own = me?.uuid === comment.user.uuid;
  const active = activeCommentID === comment.id;
  const replyLabel = replying ? 'Закрыть форму ответа' : 'Ответить';
  const style = depth > 0
    ? ({
        '--comment-indent': `${commentIndentForDepth(depth)}px`,
        '--comment-line-strength': `${Math.max(8, 30 - depth * 4)}%`,
      } as CSSProperties)
    : undefined;

  useEffect(() => {
    if (!active) return;
    ref.current?.scrollIntoView({ block: 'center', behavior: 'smooth' });
  }, [active]);

  return (
    <article ref={ref} id={`comment-${comment.id}`} className={`comment ${depth > 0 ? 'comment--reply' : ''}`} style={style}>
      <div className={`comment__content ${active ? 'comment__content--active' : ''}`}>
        <div className="comment__head">
          <UserLink user={comment.user} compact />
          <time className="comment__date" dateTime={comment.created_at}>
            {formatCommentDate(comment.created_at)}
          </time>
        </div>
        {editing ? (
          <CommentComposer
            value={text}
            onChange={setText}
            onSubmit={() => {
              onUpdate(comment.id, text);
              setEditing(false);
            }}
            placeholder="Изменить комментарий"
            submitLabel="Сохранить изменения"
            save
          />
        ) : (
          <p className={`comment__body ${comment.is_deleted ? 'comment__body--deleted muted' : ''}`}>
            {comment.is_deleted ? 'Комментарий удалён' : comment.body}
          </p>
        )}
        {!comment.is_deleted ? (
          <div className="comment__actions">
            <button
              className="comment__action-button"
              type="button"
              onClick={() => {
                haptic('light');
                setReplying((value) => !value);
              }}
              aria-label={replyLabel}
              aria-pressed={replying}
              title={replyLabel}
            >
              <Reply size={17} strokeWidth={2.3} aria-hidden="true" />
            </button>
            {own ? (
              <>
                <button
                  className="comment__action-button"
                  type="button"
                  onClick={() => {
                    haptic('light');
                    setEditing(true);
                  }}
                  aria-label="Изменить комментарий"
                  title="Изменить"
                >
                  <Pencil size={14} aria-hidden="true" />
                </button>
                <button
                  className="comment__action-button comment__action-button--danger"
                  type="button"
                  onClick={() => {
                    haptic('warning');
                    onDelete(comment.id);
                  }}
                  aria-label="Удалить комментарий"
                  title="Удалить"
                >
                  <Trash2 size={14} aria-hidden="true" />
                </button>
              </>
            ) : null}
          </div>
        ) : null}
        {replying ? (
          <ReplyForm
            onSubmit={(reply) => {
              onCreate(reply, comment.id);
              setReplying(false);
            }}
          />
        ) : null}
      </div>
      {comment.replies?.length ? (
        <div className="comment__replies">
          {comment.replies.map((reply) => (
            <CommentNode
              key={reply.id}
              comment={reply}
              activeCommentID={activeCommentID}
              me={me}
              onCreate={onCreate}
              onUpdate={onUpdate}
              onDelete={onDelete}
              depth={depth + 1}
            />
          ))}
        </div>
      ) : null}
    </article>
  );
}

function formatCommentDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';
  const now = new Date();
  const includeYear = date.getFullYear() !== now.getFullYear();
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    ...(includeYear ? { year: 'numeric' } : {}),
    hour: '2-digit',
    minute: '2-digit',
  }).format(date).replace(',', '');
}

function ReplyForm({ onSubmit }: { onSubmit: (body: string) => void }) {
  const [value, setValue] = useState('');
  return (
    <CommentComposer
      value={value}
      onChange={setValue}
      onSubmit={() => {
        onSubmit(value);
      }}
      placeholder="Ваш ответ"
      submitLabel="Отправить ответ"
    />
  );
}

function CommentComposer({
  value,
  onChange,
  onSubmit,
  placeholder,
  submitLabel,
  pending = false,
  save = false,
  root = false,
}: {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  placeholder: string;
  submitLabel: string;
  pending?: boolean;
  save?: boolean;
  root?: boolean;
}) {
  return (
    <form
      className={`comment-form ${root ? 'comment-form--root' : 'comment-form--compact'}`}
      onSubmit={(event) => {
        event.preventDefault();
        if (pending || !value.trim()) return;
        onSubmit();
      }}
    >
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        aria-label={placeholder}
        rows={1}
      />
      <button
        className="comment-form__submit"
        type="submit"
        disabled={pending || !value.trim()}
        aria-label={submitLabel}
        title={submitLabel}
        onClick={() => haptic('light')}
      >
        {pending
          ? <Loader2 className="spin" size={18} aria-hidden="true" />
          : save
            ? <Check size={19} strokeWidth={2.4} aria-hidden="true" />
            : <Send size={18} strokeWidth={2.2} aria-hidden="true" />}
      </button>
    </form>
  );
}
