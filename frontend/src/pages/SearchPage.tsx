import { Search } from 'lucide-react';
import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { TitleRow } from '../components/TitleBits';
import { EmptyState, ErrorState, LoadingState, PageHeader } from '../components/Ui';

export function SearchPage() {
  const [query, setQuery] = useState('');
  const trimmed = query.trim();
  const search = useQuery({
    queryKey: ['search', trimmed],
    queryFn: () => api.search(trimmed),
    enabled: trimmed.length >= 2,
  });

  return (
    <>
      <PageHeader title="Поиск" subtitle="Фильмы и сериалы TMDB" />
      <label className="search-box">
        <Search size={20} />
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Название фильма или сериала" />
      </label>
      {trimmed.length < 2 ? <EmptyState title="Введите запрос" text="Минимум два символа." /> : null}
      {search.isLoading ? <LoadingState label="Ищем" /> : null}
      {search.isError ? <ErrorState error={search.error} /> : null}
      {search.data ? (
        search.data.results.length ? (
          <div className="stack">
            {search.data.results.map((title) => (
              <TitleRow key={`${title.media_type}-${title.tmdb_id}`} title={title} />
            ))}
          </div>
        ) : (
          <EmptyState title="Ничего не найдено" text="Попробуйте другое название." />
        )
      ) : null}
    </>
  );
}
