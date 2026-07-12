import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { ApiError } from '../api/client';

type ToastType = 'success' | 'error' | 'warning';

type Toast = {
  id: number;
  type: ToastType;
  message: string;
};

type ToastContextValue = {
  showToast: (message: string, type?: ToastType) => void;
  showError: (error: unknown, fallback?: string) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<Toast | null>(null);
  const timer = useRef<number | undefined>(undefined);

  const showToast = useCallback((message: string, type: ToastType = 'success') => {
    window.clearTimeout(timer.current);
    setToast({ id: Date.now(), type, message });
    timer.current = window.setTimeout(() => setToast(null), 2600);
  }, []);

  const showError = useCallback(
    (error: unknown, fallback = 'Действие не выполнено') => {
      const message = error instanceof ApiError ? error.message : fallback;
      showToast(message, 'error');
    },
    [showToast],
  );

  const value = useMemo(() => ({ showToast, showError }), [showToast, showError]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-host" aria-live="polite" aria-atomic="true">
        {toast ? (
          <div key={toast.id} className={`toast toast--${toast.type}`}>
            {toast.message}
          </div>
        ) : null}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error('useToast must be used inside ToastProvider');
  }
  return value;
}
