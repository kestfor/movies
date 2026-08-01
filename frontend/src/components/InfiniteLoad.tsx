import { useEffect, useRef } from 'react';
import { Button } from './Ui';

export function InfiniteLoad({
  hasNext,
  loading,
  onLoad,
}: {
  hasNext: boolean;
  loading: boolean;
  onLoad: () => void;
}) {
  const sentinel = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const node = sentinel.current;
    if (!node || !hasNext || loading || typeof IntersectionObserver === 'undefined') return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) onLoad();
      },
      { rootMargin: '240px' },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [hasNext, loading, onLoad]);

  if (!hasNext) return null;
  return (
    <div className="infinite-load" ref={sentinel}>
      <Button variant="ghost" disabled={loading} onClick={onLoad}>
        {loading ? 'Загружаем' : 'Показать ещё'}
      </Button>
    </div>
  );
}
