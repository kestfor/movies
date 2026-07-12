import { Info, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { Criterion, RatingWithUser } from '../types/api';
import { haptic } from '../lib/telegram';
import { Accordion } from './TitleBits';
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
  const [openHint, setOpenHint] = useState<string | null>(null);
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
      <Accordion title={rating ? 'Изменить оценку' : 'Поставить оценку'} summary={`${Object.keys(scores).length}/${criteria.length}`} defaultOpen={!rating} className="rating-editor__accordion">
        <div className="sliders">
          {criteria.map((criterion) => (
            <div key={criterion.code} className={`slider-row ${openHint === criterion.code ? 'slider-row--hint-open' : ''}`}>
              <div className="slider-row__label">
                <span>{criterion.name}</span>
                {criterion.description ? (
                  <button
                    className="hint-button"
                    type="button"
                    aria-label={`Описание критерия ${criterion.name}`}
                    aria-expanded={openHint === criterion.code}
                    onClick={() => setOpenHint((current) => (current === criterion.code ? null : criterion.code))}
                  >
                    <Info size={15} />
                  </button>
                ) : null}
              </div>
              <input
                type="range"
                min="0"
                max="10"
                step="1"
                value={scores[criterion.code] || 0}
                onChange={(event) => setScore(criterion.code, event.target.value)}
              />
              <b>{scores[criterion.code] || '—'}</b>
              {criterion.description ? (
                <div className="criterion-popover" role="note">
                  <p>{criterion.description}</p>
                </div>
              ) : null}
            </div>
          ))}
        </div>
        <Button className="rating-editor__save" disabled={!Object.keys(scores).length || saving} onClick={() => onSave(scores)}>
          {saving ? 'Сохраняем' : 'Сохранить'}
        </Button>
      </Accordion>
    </section>
  );
}
