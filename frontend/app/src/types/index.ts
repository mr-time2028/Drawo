// Shared types used across features. Mirrors backend Go domains where possible.

export type UserRole = 'user' | 'admin';
export type AccountStatus = 'active' | 'banned' | 'suspended' | 'deleted' | 'inactive';

export type User = {
  id: string;
  username: string;
  avatarUrl?: string | null;
  role: UserRole;
  status: AccountStatus;
  createdAt: string;
};

export type Profile = {
  user: User;
  bio?: string | null;
  email?: string | null;
  emailVerified?: boolean;
  gamesPlayed: number;
  wins: number;
  winRate: number;
  currentStreak: number;
  bestStreak: number;
};

export type RoomType = 'public' | 'private';
export type RoomState = 'lobby' | 'playing' | 'finished' | 'closed';

export type Room = {
  id: string;
  name: string;
  inviteCode?: string | null;
  ownerId: string;
  type: RoomType;
  language: 'en' | 'fa';
  categoryId?: string | null;
  categoryName?: string | null;
  state: RoomState;
  minPlayers: number;
  maxPlayers: number;
  roundTime: number; // seconds
  maxRounds: number;
  currentRound: number;
  players: Player[];
};

export type Player = {
  id: string;
  userId: string;
  username: string;
  avatarUrl?: string | null;
  score: number;
  isDrawer: boolean;
  isHost: boolean;
  isOnline: boolean;
  isSelf?: boolean;
};

export type ChatMessageKind = 'chat' | 'guess' | 'correct' | 'system' | 'hint';

export type ChatMessage = {
  id: string;
  kind: ChatMessageKind;
  userId?: string;
  username?: string;
  text: string;
  timestamp: number;
  isSelf?: boolean;
};

// Dev/mock toggle helpers ----------------------------------------------------

export function isMockEnabled(): boolean {
  if (typeof window === 'undefined') return true; // SSR-safe default
  return new URLSearchParams(window.location.search).get('mock') !== '0';
}

export function isDevPanelEnabled(): boolean {
  if (typeof window === 'undefined') return false;
  return new URLSearchParams(window.location.search).get('dev') === '1';
}
