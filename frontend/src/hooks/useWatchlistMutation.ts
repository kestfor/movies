import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useToast } from '../components/Toast';
import { haptic } from '../lib/telegram';
import type { Title } from '../types/api';
import { updateWatchlistData } from './watchlistCache';

const watchedFamilies = ['title', 'search', 'discover', 'recommendations', 'profileWatchlist', 'watchlistMatches'];

type WatchlistVariables = { title: Title; inWatchlist: boolean };
type WatchlistContext = { snapshots: Array<[readonly unknown[], unknown]> };

export function useWatchlistMutation() {
  const queryClient = useQueryClient();
  const { showToast, showError } = useToast();

  return useMutation<void, unknown, WatchlistVariables, WatchlistContext>({
    mutationFn: async ({ title, inWatchlist }) => {
      if (inWatchlist) await api.addToWatchlist(title.media_type, title.tmdb_id);
      else await api.removeFromWatchlist(title.media_type, title.tmdb_id);
    },
    onMutate: async ({ title, inWatchlist }) => {
      const snapshots: Array<[readonly unknown[], unknown]> = [];
      for (const family of watchedFamilies) {
        await queryClient.cancelQueries({ queryKey: [family] });
        for (const [key, data] of queryClient.getQueriesData({ queryKey: [family] })) {
          snapshots.push([key, data]);
          queryClient.setQueryData(key, updateWatchlistData(data, title, inWatchlist));
        }
      }
      haptic('light');
      return { snapshots } as WatchlistContext;
    },
    onSuccess: (_, { inWatchlist }) => {
      haptic('success');
      showToast(inWatchlist ? 'Добавлено в «Хочу посмотреть»' : 'Удалено из списка');
    },
    onError: (error, _variables, context) => {
      context?.snapshots.forEach(([key, data]) => queryClient.setQueryData(key, data));
      haptic('error');
      showError(error, 'Не удалось обновить список');
    },
    onSettled: () => {
      watchedFamilies.forEach((family) => queryClient.invalidateQueries({ queryKey: [family] }));
    },
  });
}
