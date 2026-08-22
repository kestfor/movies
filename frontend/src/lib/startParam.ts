import type { MediaType } from '../types/api';

const TITLE_START_PARAM_PATTERN = /^title_(movie|tv)_([1-9][0-9]*)$/;

export function titleStartParam(mediaType: MediaType, tmdbID: number): string {
  return `title_${mediaType}_${tmdbID}`;
}

export function titleTelegramDeepLink(
  botUsername: string,
  appShortName: string,
  mediaType: MediaType,
  tmdbID: number,
): string {
  return `https://t.me/${botUsername}/${appShortName}?startapp=${titleStartParam(mediaType, tmdbID)}`;
}

export function titlePathFromStartParam(startParam: string): string | null {
  const match = startParam.match(TITLE_START_PARAM_PATTERN);
  if (!match) return null;

  return `/title/${match[1]}/${match[2]}`;
}
