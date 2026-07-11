import type { ReactNode } from 'react';
import { AlertCircle, Loader2 } from 'lucide-react';
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
  return <span className={`score-pill ${muted ? 'score-pill--muted' : ''}`}>{value ? value.toFixed(1) : '—'}</span>;
}
