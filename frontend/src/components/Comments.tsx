import { MessageCircle, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import type { Comment, User } from '../types/api';
import { Avatar } from './TitleBits';
import { Button } from './Ui';

export function Comments({
  comments,
  me,
  posting,
  onCreate,
  onUpdate,
  onDelete,
}: {
  comments: Comment[];
  me?: User;
  posting?: boolean;
  onCreate: (body: string, parentID?: number) => void;
  onUpdate: (id: number, body: string) => void;
  onDelete: (id: number) => void;
}) {
  const [body, setBody] = useState('');

  return (
    <section className="panel">
      <div className="panel__header">
        <div>
          <h2>Комментарии</h2>
          <p>{comments.length ? `${comments.length} в треде` : 'Начните обсуждение'}</p>
        </div>
        <MessageCircle size={22} />
      </div>
      <form
        className="comment-form"
        onSubmit={(event) => {
          event.preventDefault();
          if (!body.trim()) return;
          onCreate(body);
          setBody('');
        }}
      >
        <textarea value={body} onChange={(event) => setBody(event.target.value)} placeholder="Написать комментарий" />
        <Button disabled={posting || !body.trim()}>{posting ? 'Публикуем' : 'Отправить'}</Button>
      </form>
      <div className="comments-list">
        {comments.map((comment) => (
          <CommentNode
            key={comment.id}
            comment={comment}
            me={me}
            onCreate={onCreate}
            onUpdate={onUpdate}
            onDelete={onDelete}
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
}: {
  comment: Comment;
  me?: User;
  onCreate: (body: string, parentID?: number) => void;
  onUpdate: (id: number, body: string) => void;
  onDelete: (id: number) => void;
}) {
  const [replying, setReplying] = useState(false);
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(comment.body);
  const own = me?.id === comment.user.id;

  return (
    <article className={`comment ${comment.parent_id ? 'comment--reply' : ''}`}>
      <div className="comment__head">
        <Avatar name={comment.user.first_name} url={comment.user.photo_url} />
        <span>{comment.user.first_name}</span>
      </div>
      {editing ? (
        <form
          className="comment-form compact"
          onSubmit={(event) => {
            event.preventDefault();
            if (!text.trim()) return;
            onUpdate(comment.id, text);
            setEditing(false);
          }}
        >
          <textarea value={text} onChange={(event) => setText(event.target.value)} />
          <Button>Сохранить</Button>
        </form>
      ) : (
        <p className={comment.is_deleted ? 'muted' : ''}>{comment.is_deleted ? 'Комментарий удалён' : comment.body}</p>
      )}
      {!comment.is_deleted ? (
        <div className="comment__actions">
          <button type="button" onClick={() => setReplying((value) => !value)}>
            Ответить
          </button>
          {own ? (
            <>
              <button type="button" onClick={() => setEditing(true)}>
                <Pencil size={14} /> Изменить
              </button>
              <button type="button" className="danger" onClick={() => onDelete(comment.id)}>
                <Trash2 size={14} /> Удалить
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
      {comment.replies?.length ? (
        <div className="comment__replies">
          {comment.replies.map((reply) => (
            <CommentNode key={reply.id} comment={reply} me={me} onCreate={onCreate} onUpdate={onUpdate} onDelete={onDelete} />
          ))}
        </div>
      ) : null}
    </article>
  );
}

function ReplyForm({ onSubmit }: { onSubmit: (body: string) => void }) {
  const [value, setValue] = useState('');
  return (
    <form
      className="comment-form compact"
      onSubmit={(event) => {
        event.preventDefault();
        if (!value.trim()) return;
        onSubmit(value);
      }}
    >
      <textarea value={value} onChange={(event) => setValue(event.target.value)} placeholder="Ваш ответ" />
      <Button>Ответить</Button>
    </form>
  );
}
