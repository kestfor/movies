import type { Title } from '../types/api';

const activeTitleTransitionKey = 'movies.activeTitleTransition';
const activeTitleTransitionName = 'title-card-active';

type ViewTransitionDocument = Document & {
  startViewTransition?: (callback: () => void) => {
    finished: Promise<void>;
    ready: Promise<void>;
    updateCallbackDone: Promise<void>;
  };
};

export function titleTransitionName(title: Title) {
  void title;
  return activeTitleTransitionName;
}

export function titleTransitionNameFromRef(mediaType: string, tmdbID: string | number) {
  void mediaType;
  void tmdbID;
  return activeTitleTransitionName;
}

export function setActiveTitleTransition(title: Title) {
  sessionStorage.setItem(activeTitleTransitionKey, `${title.media_type}:${title.tmdb_id}`);
}

export function getActiveTitleTransitionName(title: Title) {
  const name = sessionStorage.getItem(activeTitleTransitionKey);
  return name === `${title.media_type}:${title.tmdb_id}` ? activeTitleTransitionName : undefined;
}

export function getActiveTitleTransitionNameByRef(mediaType: string, tmdbID: string | number) {
  const name = sessionStorage.getItem(activeTitleTransitionKey);
  return name === `${mediaType}:${tmdbID}` ? activeTitleTransitionName : undefined;
}

export function clearActiveTitleTransition() {
  sessionStorage.removeItem(activeTitleTransitionKey);
}

export function canUseViewTransitions() {
  return Boolean((document as ViewTransitionDocument).startViewTransition);
}

export function startViewTransition(callback: () => void) {
  const start = (document as ViewTransitionDocument).startViewTransition;
  if (!start || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    callback();
    return undefined;
  }

  return start.call(document, callback);
}
