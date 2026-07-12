import type { CSSProperties, ReactNode } from 'react';
import { AlertCircle, Loader2, Star } from 'lucide-react';
import { ApiError } from '../api/client';
import { haptic } from '../lib/telegram';

export function PageHeader({ title, subtitle, action }: { title: string; subtitle?: string; action?: ReactNode }) {
  return (
    <header className="page-header">
      <div>
        <h1>{title}</h1>
        {subtitle ? <p>{subtitle}</p> : null}
      </div>
      {action}
    </header>
  );
}

export function EmptyState({ title, text, action }: { title: string; text?: string; action?: ReactNode }) {
  return (
    <section className="state state--empty">
      <AlertCircle size={28} aria-hidden />
      <h2>{title}</h2>
      {text ? <p>{text}</p> : null}
      {action}
    </section>
  );
}

export function LoadingState({ label = 'Загрузка' }: { label?: string }) {
  return (
    <section className="state">
      <Loader2 size={28} className="spin" aria-hidden />
      <p>{label}</p>
    </section>
  );
}

export function ErrorState({ error }: { error: unknown }) {
  const message = error instanceof ApiError ? error.message : 'Что-то пошло не так';
  return <EmptyState title="Не удалось загрузить" text={message} />;
}

export function Button({
  children,
  variant = 'primary',
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'ghost' | 'danger' }) {
  return (
    <button
      {...props}
      onClick={(event) => {
        haptic('light');
        props.onClick?.(event);
      }}
      className={`button button--${variant} ${props.className || ''}`}
    >
      {children}
    </button>
  );
}

export function ScorePill({ value, muted = false }: { value?: number | null; muted?: boolean }) {
  const hasValue = typeof value === 'number' && value > 0;
  const clamped = hasValue ? Math.min(10, Math.max(1, value)) : 0;
  const hue = hasValue ? 5 + ((clamped - 1) / 9) * 135 : 0;
  const style = hasValue && !muted ? ({ '--score-hue': hue.toFixed(1) } as CSSProperties) : undefined;

  return (
    <span className={`score-pill ${muted || !hasValue ? 'score-pill--muted' : ''} ${hasValue && value >= 9 ? 'score-pill--top' : ''}`} style={style}>
      {hasValue && value >= 9 ? <Star className="score-pill__icon" size={12} fill="currentColor" aria-hidden="true" /> : null}
      {hasValue ? value.toFixed(1) : '—'}
    </span>
  );
}
