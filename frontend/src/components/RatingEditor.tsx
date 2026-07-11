import { Trash2 } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { Criterion, RatingWithUser } from '../types/api';
import { haptic } from '../lib/telegram';
import { Button } from './Ui';

export function RatingEditor({
  criteria,
  rating,
  saving,
  onSave,
  onDelete,
}: {
  criteria: Criterion[];
  rating?: RatingWithUser | null;
  saving?: boolean;
  onSave: (scores: Record<string, number>) => void;
  onDelete: () => void;
}) {
  const [scores, setScores] = useState<Record<string, number>>({});
  const lastSliderHapticAt = useRef(0);
  const avg = useMemo(() => {
    const values = Object.values(scores);
    if (!values.length) return null;
    return values.reduce((sum, value) => sum + value, 0) / values.length;
  }, [scores]);

  useEffect(() => {
    setScores(rating?.scores || {});
  }, [rating]);

  const setScore = (code: string, value: string) => {
    const parsed = Number(value);
    const now = Date.now();
    if (now - lastSliderHapticAt.current > 70) {
      haptic('light');
      lastSliderHapticAt.current = now;
    }
    setScores((current) => {
      const next = { ...current };
      if (!parsed) delete next[code];
      else next[code] = parsed;
      return next;
    });
  };

  return (
    <section className="panel">
      <div className="panel__header">
        <div>
          <h2>Моя оценка</h2>
          <p>{avg ? `Итог ${avg.toFixed(1)}` : 'Заполните хотя бы один критерий'}</p>
        </div>
        {rating ? (
          <button
            className="icon-button danger"
            type="button"
            onClick={() => {
              haptic('light');
              onDelete();
            }}
            aria-label="Удалить оценку"
          >
            <Trash2 size={18} />
          </button>
        ) : null}
      </div>
      <div className="sliders">
        {criteria.map((criterion) => (
          <label key={criterion.code} className="slider-row">
            <span>{criterion.name}</span>
            <input
              type="range"
              min="0"
              max="10"
              step="1"
              value={scores[criterion.code] || 0}
              onChange={(event) => setScore(criterion.code, event.target.value)}
            />
            <b>{scores[criterion.code] || '—'}</b>
          </label>
        ))}
      </div>
      <Button disabled={!Object.keys(scores).length || saving} onClick={() => onSave(scores)}>
        {saving ? 'Сохраняем' : 'Сохранить оценку'}
      </Button>
    </section>
  );
}
