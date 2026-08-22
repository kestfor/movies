import type { MediaType } from '../types/api';
import { titleTelegramDeepLink } from './startParam';

type ThemeParams = Record<string, string | undefined>;

type TelegramWebApp = {
  initData: string;
  initDataUnsafe?: {
    start_param?: string;
    user?: {
      id: number;
      first_name?: string;
      username?: string;
      photo_url?: string;
    };
  };
  themeParams?: ThemeParams;
  colorScheme?: 'light' | 'dark';
  ready?: () => void;
  expand?: () => void;
  close?: () => void;
  openTelegramLink?: (url: string) => void;
  HapticFeedback?: {
    impactOccurred?: (style: 'light' | 'medium' | 'heavy') => void;
    notificationOccurred?: (type: 'error' | 'success' | 'warning') => void;
  };
  BackButton?: {
    show: () => void;
    hide: () => void;
    onClick: (handler: () => void) => void;
    offClick: (handler: () => void) => void;
  };
};

declare global {
  interface Window {
    Telegram?: {
      WebApp?: TelegramWebApp;
    };
  }
}

export const tg = window.Telegram?.WebApp;

const DEV_BOT_TOKEN = 'dev-bot-token';
const DEV_TG_ID = 111;
const DEV_USERNAME = 'ivan';
const DEV_FIRST_NAME = 'Иван';
const DEV_PHOTO_URL = 'https://example.com/photo.jpg';
const DEV_INIT_DATA_MAX_AGE_SECONDS = 50 * 60;

function isLocalHost(): boolean {
  return ['localhost', '127.0.0.1', '0.0.0.0'].includes(window.location.hostname);
}

async function buildDevInitData(): Promise<string> {
  const cached = localStorage.getItem('movies.dev_init_data');
  if (cached && !isExpiredDevInitData(cached)) return cached;

  const authDate = String(Math.floor(Date.now() / 1000));
  const user = JSON.stringify(
    {
      id: DEV_TG_ID,
      username: DEV_USERNAME,
      first_name: DEV_FIRST_NAME,
      photo_url: DEV_PHOTO_URL,
    },
    null,
  );
  const values: Record<string, string> = {
    auth_date: authDate,
    query_id: 'AAEAAAE',
    user,
  };
  const dataCheck = Object.keys(values)
    .sort()
    .map((key) => `${key}=${values[key]}`)
    .join('\n');

  const encoder = new TextEncoder();
  const botToken = import.meta.env.VITE_DEV_BOT_TOKEN || DEV_BOT_TOKEN;
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode('WebAppData'),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const secret = await crypto.subtle.sign('HMAC', keyMaterial, encoder.encode(botToken));
  const signatureKey = await crypto.subtle.importKey(
    'raw',
    secret,
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const hashBuffer = await crypto.subtle.sign('HMAC', signatureKey, encoder.encode(dataCheck));
  const hash = Array.from(new Uint8Array(hashBuffer), (byte) => byte.toString(16).padStart(2, '0')).join('');
  const initData = `auth_date=${encodeURIComponent(authDate)}&query_id=${encodeURIComponent('AAEAAAE')}&user=${encodeURIComponent(user)}&hash=${hash}`;

  localStorage.setItem('movies.dev_init_data', initData);
  return initData;
}

function isExpiredDevInitData(initData: string): boolean {
  const authDate = Number(new URLSearchParams(initData).get('auth_date') || 0);
  if (!authDate) return true;

  const now = Math.floor(Date.now() / 1000);
  return now - authDate > DEV_INIT_DATA_MAX_AGE_SECONDS;
}

export async function getInitData(): Promise<string> {
  if (tg?.initData) return tg.initData;
  const envInitData = import.meta.env.VITE_DEV_INIT_DATA;
  if (envInitData) return envInitData;
  if (isLocalHost()) return buildDevInitData();
  return '';
}

export function getStartParam(): string {
  const unsafeParam = tg?.initDataUnsafe?.start_param;
  if (unsafeParam) return unsafeParam;

  const params = new URLSearchParams(window.location.search);
  return params.get('tgWebAppStartParam') || '';
}

export function getInviteUserUUID(): string | null {
  const match = getStartParam().match(/^uid_([0-9a-fA-F-]{36})$/);
  return match ? match[1] : null;
}

export function bootTelegram(): void {
  tg?.ready?.();
  tg?.expand?.();
  applyTelegramTheme();
}

export function applyTelegramTheme(): void {
  const theme = tg?.themeParams || {};
  const root = document.documentElement;
  const variables: Record<string, string | undefined> = {
    '--tg-bg': theme.bg_color,
    '--tg-text': theme.text_color,
    '--tg-hint': theme.hint_color,
    '--tg-link': theme.link_color,
    '--tg-button': theme.button_color,
    '--tg-button-text': theme.button_text_color,
    '--tg-secondary-bg': theme.secondary_bg_color,
    '--tg-header-bg': theme.header_bg_color,
    '--tg-accent-text': theme.accent_text_color,
  };

  Object.entries(variables).forEach(([key, value]) => {
    if (value) root.style.setProperty(key, value);
  });

  root.dataset.theme = tg?.colorScheme || 'dark';
}

type HapticType = 'light' | 'medium' | 'heavy' | 'success' | 'warning' | 'error';

export function haptic(type: HapticType = 'light'): void {
  if (type === 'light' || type === 'medium' || type === 'heavy') {
    tg?.HapticFeedback?.impactOccurred?.(type);
    return;
  }
  tg?.HapticFeedback?.notificationOccurred?.(type);
}

export function shareInvite(currentUserUUID: string): void {
  haptic('light');
  const bot = import.meta.env.VITE_BOT_USERNAME || 'moviesclubtechbot';
  const app = import.meta.env.VITE_WEBAPP_SHORT_NAME || 'moviesclub';
  const text = encodeURIComponent('Добавляйся в мой КиноКруг');
  const link = bot && app
    ? `https://t.me/${bot}/${app}?startapp=uid_${currentUserUUID}`
    : `${window.location.origin}/friends?invite=uid_${currentUserUUID}`;
  const shareUrl = `https://t.me/share/url?url=${encodeURIComponent(link)}&text=${text}`;
  tg?.openTelegramLink?.(shareUrl) || window.open(shareUrl, '_blank');
}

export function shareTitle(mediaType: MediaType, tmdbID: number, title: string): void {
  haptic('light');
  const bot = import.meta.env.VITE_BOT_USERNAME || 'moviesclubtechbot';
  const app = import.meta.env.VITE_WEBAPP_SHORT_NAME || 'moviesclub';
  const link = titleTelegramDeepLink(bot, app, mediaType, tmdbID);
  const text = encodeURIComponent(`Смотри «${title}» в КиноКруге`);
  const shareUrl = `https://t.me/share/url?url=${encodeURIComponent(link)}&text=${text}`;

  if (tg?.openTelegramLink) {
    tg.openTelegramLink(shareUrl);
    return;
  }
  window.open(shareUrl, '_blank');
}
