import { Navigate, Outlet } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { ErrorState, LoadingState } from '../components/Ui';

export function AuthGate() {
  const me = useQuery({ queryKey: ['me'], queryFn: api.me, retry: false });

  if (me.isLoading) return <LoadingState label="Входим" />;
  if (me.isError) return <ErrorState error={me.error} />;

  return me.data ? <Outlet /> : <Navigate to="/feed" replace />;
}
