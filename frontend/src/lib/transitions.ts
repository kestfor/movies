import type { Title } from '../types/api';

const activeTitleTransitionKey = 'movies.activeTitleTransition';
const suppressPageTransitionKey = 'movies.suppressNextPageTransition';
const preserveScrollKey = 'movies.preserveNextScroll';
const activeTitleTransitionName = 'title-card-active';

export type TitleTransitionSnapshot = Pick<
  Title,
  'media_type' | 'tmdb_id' | 'title' | 'original_title' | 'poster_path' | 'release_year' | 'genres' | 'overview'
>;

export type ActiveTitleTransition = {
  ref: string;
  sourcePath: string;
  sourceScrollY: number;
  title: TitleTransitionSnapshot;
};

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
  const transition: ActiveTitleTransition = {
    ref: titleRef(title.media_type, title.tmdb_id),
    sourcePath: `${window.location.pathname}${window.location.search}`,
    sourceScrollY: window.scrollY,
    title: {
      media_type: title.media_type,
      tmdb_id: title.tmdb_id,
      title: title.title,
      original_title: title.original_title,
      poster_path: title.poster_path,
      release_year: title.release_year,
      genres: title.genres,
      overview: title.overview,
    },
  };
  sessionStorage.setItem(activeTitleTransitionKey, JSON.stringify(transition));
}

export function getActiveTitleTransitionName(title: Title) {
  return isActiveTitleRef(title.media_type, title.tmdb_id) ? activeTitleTransitionName : undefined;
}

export function getActiveTitleTransitionNameByRef(mediaType: string, tmdbID: string | number) {
  return isActiveTitleRef(mediaType, tmdbID) ? activeTitleTransitionName : undefined;
}

export function getActiveTitleTransitionByRef(mediaType: string, tmdbID: string | number) {
  const transition = getActiveTitleTransition();
  return transition?.ref === titleRef(mediaType, tmdbID) ? transition : undefined;
}

export function getActiveTitleTransition() {
  const raw = sessionStorage.getItem(activeTitleTransitionKey);
  if (!raw) return undefined;
  try {
    return JSON.parse(raw) as ActiveTitleTransition;
  } catch {
    sessionStorage.removeItem(activeTitleTransitionKey);
    return undefined;
  }
}

export function clearActiveTitleTransition() {
  sessionStorage.removeItem(activeTitleTransitionKey);
}

export function suppressNextPageTransition() {
  sessionStorage.setItem(suppressPageTransitionKey, '1');
}

export function consumeSuppressPageTransition() {
  const value = sessionStorage.getItem(suppressPageTransitionKey) === '1';
  if (value) sessionStorage.removeItem(suppressPageTransitionKey);
  return value;
}

export function preserveNextScroll() {
  sessionStorage.setItem(preserveScrollKey, '1');
}

export function consumePreserveScroll() {
  const value = sessionStorage.getItem(preserveScrollKey) === '1';
  if (value) sessionStorage.removeItem(preserveScrollKey);
  return value;
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

export function titleRef(mediaType: string, tmdbID: string | number) {
  return `${mediaType}:${tmdbID}`;
}

function isActiveTitleRef(mediaType: string, tmdbID: string | number) {
  const transition = getActiveTitleTransition();
  return transition?.ref === titleRef(mediaType, tmdbID);
}
