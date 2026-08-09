import type { User, WatchlistMatchItem } from '../types/api';
import { watchTogetherPeopleLabel } from '../lib/watchTogether';
import { Avatar, TitleRow } from './TitleBits';

export function WatchTogetherCard({ item }: { item: WatchlistMatchItem }) {
  return (
    <article className="watch-together-card">
      <TitleRow
        title={item.title}
        footer={
          <div className="watch-together-card__people">
            <span className="watch-together-card__avatars" aria-hidden>
              {item.users.slice(0, 4).map((user) => <Avatar key={user.uuid} name={user.first_name} url={user.photo_url} />)}
              {item.users.length > 4 ? <span className="avatar watch-together-card__more">+{item.users.length - 4}</span> : null}
            </span>
            <span className="watch-together-card__label">{watchTogetherPeopleLabel(item.users)}</span>
          </div>
        }
      />
    </article>
  );
}

export function FriendAvatarFilter({
  friends,
  selected,
  onToggle,
  onClear,
}: {
  friends: User[];
  selected: readonly string[];
  onToggle: (uuid: string) => void;
  onClear: () => void;
}) {
  const selectedSet = new Set(selected);
  return (
    <section className="people-filter" aria-label="Фильтр по друзьям">
      <div className="people-filter__header">
        <span>С кем смотреть</span>
        {selected.length ? <button type="button" onClick={onClear}>Сбросить</button> : <span className="muted">Все совпадения</span>}
      </div>
      <div className="people-filter__list">
        {friends.map((friend) => {
          const active = selectedSet.has(friend.uuid);
          return (
            <button
              key={friend.uuid}
              className={`people-filter__person ${active ? 'is-active' : ''}`}
              type="button"
              aria-pressed={active}
              onClick={() => onToggle(friend.uuid)}
            >
              <span className="people-filter__avatar">
                <Avatar name={friend.first_name} url={friend.photo_url} />
                {active ? <span className="people-filter__check" aria-hidden>✓</span> : null}
              </span>
              <span>{friend.first_name}</span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
