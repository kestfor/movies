import { Sparkles, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, KeyboardEvent, PointerEvent as ReactPointerEvent, TransitionEvent } from 'react';
import type { Criterion, RatingWithUser } from '../types/api';
import { haptic } from '../lib/telegram';
import { Accordion } from './TitleBits';
import { Button } from './Ui';

type RatingMood = {
  emoji: string;
  label: string;
  hue: number;
};

type DragState = {
  pointerID: number;
  startX: number;
  startY: number;
  lastX: number;
  lastAt: number;
  velocityX: number;
  axis: 'pending' | 'horizontal' | 'vertical';
  width: number;
};

const SWIPE_AXIS_THRESHOLD = 8;
const SWIPE_HORIZONTAL_DOMINANCE = 1.05;
const SWIPE_VERTICAL_DOMINANCE = 1.25;
const SWIPE_DISTANCE_RATIO = 0.25;
const SWIPE_VELOCITY_THRESHOLD = 0.5;

const EMPTY_CRITERION_EMOJIS: Record<string, string> = {
  story: '📖',
  plot: '📖',
  characters: '👥',
  acting: '🎭',
  direction: '🎬',
  directing: '🎬',
  visuals: '🎨',
  sound: '🎵',
  music: '🎵',
  atmosphere: '🌌',
};

function getRatingMood(value?: number | null, criterionCode?: string): RatingMood {
  if (!value) return { emoji: criterionCode ? EMPTY_CRITERION_EMOJIS[criterionCode] || '🎞️' : '✨', label: 'Без оценки', hue: 210 };
  if (value <= 3) return { emoji: '😬', label: 'Совсем не зашло', hue: 5 };
  if (value <= 5) return { emoji: '😐', label: 'Скорее мимо', hue: 34 };
  if (value <= 7) return { emoji: '🙂', label: 'Хорошо', hue: 84 };
  if (value <= 9) return { emoji: '🤩', label: 'Очень зашло', hue: 145 };
  return { emoji: '🏆', label: 'Идеально!', hue: 48 };
}

function moodStyle(value?: number | null) {
  return { '--rating-hue': getRatingMood(value).hue } as CSSProperties;
}

export function RatingEditor({
  criteria,
  rating,
  saving,
  celebrating,
  open,
  onOpenChange,
  onSave,
  onDelete,
}: {
  criteria: Criterion[];
  rating?: RatingWithUser | null;
  saving?: boolean;
  celebrating?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onSave: (scores: Record<string, number>) => void;
  onDelete: () => void;
}) {
  const [scores, setScores] = useState<Record<string, number>>({});
  const [activeIndex, setActiveIndex] = useState(rating ? criteria.length : 0);
  const filledCount = Object.keys(scores).length;
  const avg = useMemo(() => {
    const values = Object.values(scores);
    return values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null;
  }, [scores]);

  useEffect(() => {
    setScores(rating?.scores || {});
    setActiveIndex(rating ? criteria.length : 0);
  }, [criteria, rating]);

  const selectCard = (index: number) => {
    const nextIndex = Math.min(criteria.length, Math.max(0, index));
    if (nextIndex !== activeIndex) haptic('light');
    setActiveIndex(nextIndex);
  };

  const setScore = (code: string, value?: number) => {
    setScores((current) => {
      const next = { ...current };
      if (value === undefined) delete next[code];
      else next[code] = value;
      return next;
    });
  };

  return (
    <section className="panel rating-editor">
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
              haptic('warning');
              onDelete();
            }}
            aria-label="Удалить оценку"
          >
            <Trash2 size={18} />
          </button>
        ) : null}
      </div>
      <Accordion
        title={rating ? 'Изменить оценку' : 'Поставить оценку'}
        summary={`${filledCount}/${criteria.length}`}
        defaultOpen={!rating}
        open={open}
        onOpenChange={onOpenChange}
        className="rating-editor__accordion"
      >
        <RatingCarousel
          criteria={criteria}
          scores={scores}
          avg={avg}
          saving={saving}
          celebrating={celebrating}
          activeIndex={activeIndex}
          onIndexChange={selectCard}
          onScoreChange={setScore}
          onSave={() => onSave(scores)}
        />
      </Accordion>
    </section>
  );
}

function RatingCarousel({
  criteria,
  scores,
  avg,
  saving,
  celebrating,
  activeIndex,
  onIndexChange,
  onScoreChange,
  onSave,
}: {
  criteria: Criterion[];
  scores: Record<string, number>;
  avg: number | null;
  saving?: boolean;
  celebrating?: boolean;
  activeIndex: number;
  onIndexChange: (index: number) => void;
  onScoreChange: (code: string, value?: number) => void;
  onSave: () => void;
}) {
  const drag = useRef<DragState | null>(null);
  const pendingIndex = useRef<number | null>(null);
  const [dragOffset, setDragOffset] = useState(0);
  const [dragDirection, setDragDirection] = useState<1 | -1>(1);
  const [deckWidth, setDeckWidth] = useState(1);
  const [dragging, setDragging] = useState(false);
  const [settling, setSettling] = useState(false);
  const totalCards = criteria.length + 1;
  const summaryIndex = criteria.length;

  const finishSettling = () => {
    if (!settling) return;
    const nextIndex = pendingIndex.current;
    pendingIndex.current = null;
    setSettling(false);
    if (nextIndex !== null) onIndexChange(nextIndex);
    setDragOffset(0);
  };

  useEffect(() => {
    if (!settling) return undefined;
    const timer = window.setTimeout(finishSettling, 420);
    return () => window.clearTimeout(timer);
  }, [settling]);

  const beginDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (settling || event.pointerType === 'mouse' && event.button !== 0) return;
    const target = event.target;
    if (target instanceof Element && target.closest('button, a, input, textarea, select, [role="slider"]')) return;

    const width = event.currentTarget.clientWidth;
    setDeckWidth(width);
    drag.current = {
      pointerID: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      lastX: event.clientX,
      lastAt: event.timeStamp,
      velocityX: 0,
      axis: 'pending',
      width,
    };
    event.currentTarget.setPointerCapture(event.pointerId);
  };

  const moveDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    const current = drag.current;
    if (!current || current.pointerID !== event.pointerId) return;

    const dx = event.clientX - current.startX;
    const dy = event.clientY - current.startY;
    if (current.axis === 'pending') {
      if (Math.hypot(dx, dy) < SWIPE_AXIS_THRESHOLD) return;

      const absX = Math.abs(dx);
      const absY = Math.abs(dy);
      if (absX >= absY * SWIPE_HORIZONTAL_DOMINANCE) {
        current.axis = 'horizontal';
      } else if (absY >= absX * SWIPE_VERTICAL_DOMINANCE) {
        current.axis = 'vertical';
        return;
      } else {
        return;
      }

      setDragging(true);
    }
    if (current.axis !== 'horizontal') return;

    event.preventDefault();
    const elapsed = Math.max(1, event.timeStamp - current.lastAt);
    current.velocityX = (event.clientX - current.lastX) / elapsed;
    current.lastX = event.clientX;
    current.lastAt = event.timeStamp;

    const direction = dx < 0 ? 1 : -1;
    setDragDirection(direction);
    const atStart = activeIndex === 0 && direction === -1;
    const atEnd = activeIndex === totalCards - 1 && direction === 1;
    setDragOffset(atStart || atEnd ? dx * 0.24 : dx);
  };

  const endDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    const current = drag.current;
    if (!current || current.pointerID !== event.pointerId) return;
    drag.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);

    if (current.axis !== 'horizontal') {
      setDragging(false);
      setDragOffset(0);
      return;
    }

    const dx = event.clientX - current.startX;
    const directionSource = Math.abs(current.velocityX) >= SWIPE_VELOCITY_THRESHOLD ? current.velocityX : dx;
    const nextDirection = directionSource < 0 ? 1 : -1;
    const nextIndex = activeIndex + nextDirection;
    const withinBounds = nextIndex >= 0 && nextIndex < totalCards;
    const crossedDistance = Math.abs(dx) >= current.width * SWIPE_DISTANCE_RATIO;
    const crossedVelocity = Math.abs(current.velocityX) >= SWIPE_VELOCITY_THRESHOLD;
    pendingIndex.current = withinBounds && (crossedDistance || crossedVelocity) ? nextIndex : null;
    setDragDirection(nextDirection);
    setDragging(false);
    setSettling(true);
    setDragOffset(pendingIndex.current === null ? 0 : nextDirection * -current.width * 1.16);
  };

  const cancelDrag = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!drag.current || drag.current.pointerID !== event.pointerId) return;
    drag.current = null;
    pendingIndex.current = null;
    setDragging(false);
    setSettling(true);
    setDragOffset(0);
  };

  const progress = Math.min(1, Math.abs(dragOffset) / Math.max(1, deckWidth));
  const defaultDirection: 1 | -1 = activeIndex < totalCards - 1 ? 1 : -1;
  const revealDirection = dragOffset ? dragDirection : defaultDirection;
  const revealedIndex = activeIndex === summaryIndex && !dragOffset ? -1 : activeIndex + revealDirection;
  const activeMoodValue = activeIndex === summaryIndex ? avg : scores[criteria[activeIndex]?.code];

  const getDeckCardStyle = (index: number): CSSProperties => {
    if (index === activeIndex) {
      if (index === summaryIndex) {
        return {
          zIndex: 3,
          opacity: 1 - progress * 0.12,
          transform: `translate3d(${dragOffset}px, 0, 0) scale(${1 - progress * 0.025})`,
        };
      }
      const rotation = (dragOffset / Math.max(1, deckWidth)) * 6;
      return {
        zIndex: 3,
        opacity: 1,
        transform: `translate3d(${dragOffset}px, 0, 0) rotate(${rotation}deg) scale(${1 - progress * 0.035})`,
      };
    }
    if (index === revealedIndex) {
      if (index === summaryIndex || activeIndex === summaryIndex) {
        return {
          zIndex: 2,
          opacity: progress,
          transform: `translate3d(0, ${8 * (1 - progress)}px, 0) scale(${0.95 + progress * 0.05})`,
        };
      }
      const lift = 12 * (1 - progress);
      const scale = 0.955 + progress * 0.045;
      return {
        zIndex: 2,
        opacity: 0.72 + progress * 0.28,
        transform: `translate3d(0, ${lift}px, 0) rotate(${revealDirection * -0.7}deg) scale(${scale})`,
      };
    }
    return {
      zIndex: 1,
      opacity: 0,
      transform: 'translate3d(0, 18px, 0) scale(0.93)',
    };
  };

  const finishOnActiveTransition = (event: TransitionEvent<HTMLDivElement>) => {
    if (event.propertyName === 'transform' && event.currentTarget.classList.contains('is-active')) finishSettling();
  };

  return (
    <div className="rating-journey" style={moodStyle(activeMoodValue)}>
      <div className="rating-journey__topline">
        <span>{activeIndex === summaryIndex ? 'Итоговая оценка' : `Критерий ${activeIndex + 1} из ${criteria.length}`}</span>
        <button
          className={activeIndex === summaryIndex ? 'is-active' : ''}
          type="button"
          onClick={() => onIndexChange(summaryIndex)}
          aria-current={activeIndex === summaryIndex ? 'step' : undefined}
        >
          <Sparkles size={14} aria-hidden="true" />
          Итог {avg ? avg.toFixed(1) : '—'}
        </button>
      </div>
      <div className="rating-progress" aria-label="Критерии оценки">
        {criteria.map((criterion, index) => (
          <button
            key={criterion.code}
            className={`${index === activeIndex ? 'is-active' : ''} ${scores[criterion.code] ? 'is-filled' : ''}`}
            type="button"
            onClick={() => onIndexChange(index)}
            aria-label={`${criterion.name}: ${scores[criterion.code] ? `${scores[criterion.code]} из 10` : 'не оценено'}`}
            aria-current={index === activeIndex ? 'step' : undefined}
          >
            <span />
          </button>
        ))}
      </div>
      <div
        className={`rating-carousel rating-deck ${dragging ? 'is-dragging' : ''} ${settling ? 'is-settling' : ''}`}
        onPointerDown={beginDrag}
        onPointerMove={moveDrag}
        onPointerUp={endDrag}
        onPointerCancel={cancelDrag}
        onLostPointerCapture={cancelDrag}
      >
        {criteria.map((criterion, index) => {
          const active = index === activeIndex;
          const revealed = index === revealedIndex;
          return (
            <div
              key={criterion.code}
              className={`rating-deck-card ${active ? 'is-active' : ''} ${revealed ? 'is-revealed' : ''}`}
              style={getDeckCardStyle(index)}
              aria-hidden={!active}
              onTransitionEnd={active ? finishOnActiveTransition : undefined}
            >
              <RatingCriterionCard
                criterion={criterion}
                value={scores[criterion.code]}
                active={active}
                onChange={(value) => onScoreChange(criterion.code, value)}
              />
            </div>
          );
        })}
        <div
          className={`rating-deck-card ${activeIndex === summaryIndex ? 'is-active' : ''} ${revealedIndex === summaryIndex ? 'is-revealed' : ''}`}
          style={getDeckCardStyle(summaryIndex)}
          aria-hidden={activeIndex !== summaryIndex}
          onTransitionEnd={activeIndex === summaryIndex ? finishOnActiveTransition : undefined}
        >
          <RatingSummary
            criteria={criteria}
            scores={scores}
            avg={avg}
            saving={saving}
            celebrating={celebrating}
            active={activeIndex === summaryIndex}
            onEdit={onIndexChange}
            onSave={onSave}
          />
        </div>
      </div>
    </div>
  );
}

function RatingCriterionCard({
  criterion,
  value,
  active,
  onChange,
}: {
  criterion: Criterion;
  value?: number;
  active: boolean;
  onChange: (value?: number) => void;
}) {
  const mood = getRatingMood(value, criterion.code);

  return (
    <article className="rating-criterion-card" style={moodStyle(value)}>
      <div key={`${criterion.code}-${value ?? 'empty'}`} className="rating-criterion-card__emoji" aria-hidden="true">
        {mood.emoji}
      </div>
      <h3>{criterion.name}</h3>
      {criterion.description ? <p className="rating-criterion-card__description">{criterion.description}</p> : null}
      <div className={`rating-criterion-card__score ${value ? 'has-value' : ''}`}>
        {value ? <>{value}<small>/10</small></> : '—'}
      </div>
      <p className="rating-criterion-card__reaction">{mood.label}</p>
      <ScoreSlider criterion={criterion.name} value={value} active={active} onChange={onChange} />
    </article>
  );
}

function ScoreSlider({
  criterion,
  value,
  active,
  onChange,
}: {
  criterion: string;
  value?: number;
  active: boolean;
  onChange: (value?: number) => void;
}) {
  const activePointer = useRef<number | null>(null);
  const lastReportedValue = useRef(value);

  useEffect(() => {
    lastReportedValue.current = value;
  }, [value]);

  const reportValue = (nextValue?: number) => {
    if (lastReportedValue.current === nextValue) return;
    lastReportedValue.current = nextValue;
    haptic('light');
    onChange(nextValue);
  };

  const reportFromX = (element: HTMLDivElement, clientX: number) => {
    const rail = element.querySelector<HTMLElement>('.rating-slider__rail');
    const bounds = (rail || element).getBoundingClientRect();
    const ratio = Math.min(1, Math.max(0, (clientX - bounds.left) / bounds.width));
    const position = Math.round(ratio * 10);
    reportValue(position === 0 ? undefined : position);
  };

  const changeFromKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    let nextPosition: number | null = null;
    const keyboardPosition = value ?? 0;
    if (event.key === 'ArrowLeft' || event.key === 'ArrowDown') nextPosition = Math.max(0, keyboardPosition - 1);
    if (event.key === 'ArrowRight' || event.key === 'ArrowUp') nextPosition = Math.min(10, keyboardPosition + 1);
    if (event.key === 'Home') nextPosition = 0;
    if (event.key === 'End') nextPosition = 10;
    if (nextPosition === null || nextPosition === keyboardPosition) return;
    event.preventDefault();
    reportValue(nextPosition === 0 ? undefined : nextPosition);
  };

  const progress = ((value ?? 0) / 10) * 100;

  return (
    <div
      className={`rating-slider ${value ? 'has-value' : 'is-empty'}`}
      style={{ '--score-progress': `${progress}%` } as CSSProperties}
      role="slider"
      tabIndex={active ? 0 : -1}
      aria-label={`Оценка критерия «${criterion}»`}
      aria-valuemin={0}
      aria-valuemax={10}
      aria-valuenow={value ?? 0}
      aria-valuetext={value ? `${value} из 10` : 'Не оценено'}
      onKeyDown={changeFromKeyboard}
      onPointerDown={(event) => {
        event.stopPropagation();
        event.preventDefault();
        activePointer.current = event.pointerId;
        event.currentTarget.setPointerCapture(event.pointerId);
        reportFromX(event.currentTarget, event.clientX);
      }}
      onPointerMove={(event) => {
        if (activePointer.current !== event.pointerId) return;
        event.stopPropagation();
        event.preventDefault();
        reportFromX(event.currentTarget, event.clientX);
      }}
      onPointerUp={(event) => {
        if (activePointer.current !== event.pointerId) return;
        activePointer.current = null;
        if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
      }}
      onPointerCancel={(event) => {
        if (activePointer.current === event.pointerId) activePointer.current = null;
      }}
    >
      <div className="rating-slider__rail">
        <span className="rating-slider__fill" />
        <span className="rating-slider__thumb" />
      </div>
      <div className="rating-slider__ticks" aria-hidden="true">
        {Array.from({ length: 11 }, (_, index) => <span key={index}>{index || '—'}</span>)}
      </div>
    </div>
  );
}

function RatingSummary({
  criteria,
  scores,
  avg,
  saving,
  celebrating,
  active,
  onEdit,
  onSave,
}: {
  criteria: Criterion[];
  scores: Record<string, number>;
  avg: number | null;
  saving?: boolean;
  celebrating?: boolean;
  active: boolean;
  onEdit: (index: number) => void;
  onSave: () => void;
}) {
  const mood = getRatingMood(avg);

  return (
    <article className={`rating-summary ${celebrating ? 'is-celebrating' : ''}`} style={moodStyle(avg)}>
      <div key={avg?.toFixed(1) || 'empty'} className="rating-summary__emoji" aria-hidden="true">{mood.emoji}</div>
      <span className="rating-summary__eyebrow">Ваша итоговая оценка</span>
      <div className="rating-summary__score">{avg ? avg.toFixed(1) : '—'}{avg ? <small>/10</small> : null}</div>
      <p>{avg ? mood.label : 'Выберите хотя бы один критерий'}</p>
      <div className="rating-summary__criteria">
        {criteria.map((criterion, index) => (
          <button key={criterion.code} type="button" onClick={() => onEdit(index)} tabIndex={active ? 0 : -1}>
            <span>{criterion.name}</span>
            <b>{scores[criterion.code] || '—'}</b>
          </button>
        ))}
      </div>
      <div className="rating-save-celebration">
        <Button className="rating-editor__save" disabled={!Object.keys(scores).length || saving || celebrating} tabIndex={active ? 0 : -1} onClick={onSave}>
          {saving ? 'Сохраняем' : celebrating ? 'Готово!' : 'Сохранить оценку'}
        </Button>
        <span className="rating-save-celebration__particles" aria-hidden="true">
          {Array.from({ length: 8 }, (_, index) => <i key={index} />)}
        </span>
      </div>
    </article>
  );
}
