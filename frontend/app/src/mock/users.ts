import type { Profile, User } from '@/types';

const base = (overrides: Partial<User> = {}): User => ({
  id: overrides.id || 'u-0',
  username: overrides.username || 'drawo',
  avatarUrl: overrides.avatarUrl ?? null,
  role: overrides.role || 'user',
  status: overrides.status || 'active',
  createdAt: overrides.createdAt || '2026-01-01T00:00:00Z',
});

export const mockMe: User = base({
  id: 'me',
  username: 'you',
  role: 'admin', // makes admin entry visible in design phase; toggle to 'user' to see non-admin
});

export const mockMeProfile: Profile = {
  user: mockMe,
  bio: 'Drawing cats since 2024 🎨',
  email: 'you@drawo.app',
  emailVerified: true,
  gamesPlayed: 127,
  wins: 42,
  winRate: 0.33,
  currentStreak: 3,
  bestStreak: 9,
};

export const mockUsers: User[] = [
  base({ id: 'u1', username: 'Alice', avatarUrl: null }),
  base({ id: 'u2', username: 'Babak', avatarUrl: null }),
  base({ id: 'u3', username: 'Charlie', avatarUrl: null }),
  base({ id: 'u4', username: 'Dina', avatarUrl: null }),
  base({ id: 'u5', username: 'Ehsan', avatarUrl: null }),
  base({ id: 'u6', username: 'Faezeh', avatarUrl: null }),
  base({ id: 'u7', username: 'Golnaz', avatarUrl: null }),
  base({ id: 'u8', username: 'Hossein', role: 'admin', avatarUrl: null }),
  base({ id: 'u9', username: 'BannedUser', status: 'banned', avatarUrl: null }),
];

/**
 * Generate an SVG data-URL avatar from a username (deterministic color + initial).
 * Used as avatar fallback everywhere during the design phase.
 */
export function avatarFor(name: string): string {
  const colors = ['#4A98F7', '#22C55E', '#F97316', '#EF4444', '#A855F7', '#14B8A6', '#F59E0B', '#EC4899'];
  const letter = (name.trim()[0] || '?').toUpperCase();
  const idx = [...name].reduce((acc, ch) => acc + ch.charCodeAt(0), 0) % colors.length;
  const bg = colors[idx];
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="16" fill="${bg}"/><text x="50%" y="54%" text-anchor="middle" dominant-baseline="middle" font-family="Inter, system-ui, sans-serif" font-weight="800" font-size="30" fill="#fff">${letter}</text></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}
