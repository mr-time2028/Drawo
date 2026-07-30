import type { AccountStatus, UserRole } from '@/types';
import { avatarFor } from './users';

export type AdminUser = {
  id: string;
  username: string;
  avatarUrl: string;
  role: UserRole;
  status: AccountStatus;
  joinedAt: string;
  gamesPlayed: number;
  email?: string;
};

export type AdminReport = {
  id: string;
  reporter: string;
  target: string;
  reason: string;
  status: 'pending' | 'confirmed' | 'rejected';
  createdAt: string;
};

export type AdminCategory = { id: string; nameEn: string; nameFa: string; wordCount: number };
export type AdminWord = {
  id: string;
  text: string;
  language: 'en' | 'fa';
  categoryId: string;
  difficulty: 'easy' | 'medium' | 'hard';
};
export type AdminBadWord = { id: string; text: string; language: 'en' | 'fa' };
export type AdminSong = {
  id: string;
  title: string;
  durationSec: number;
  enabled: boolean;
};

const rand = (min: number, max: number) => Math.floor(min + Math.random() * (max - min));
const dateOffset = (days: number) => new Date(Date.now() - days * 86400_000).toISOString();

const userNames = [
  'Alice',
  'Babak',
  'Charlie',
  'Dina',
  'Ehsan',
  'Faezeh',
  'Golnaz',
  'Hossein',
  'Iman',
  'Jaleh',
  'Kamran',
  'Leila',
  'Mina',
  'Navid',
  'Omid',
  'Parisa',
  'Qasem',
  'Roya',
  'Saman',
  'Tara',
];
const statuses: AccountStatus[] = [
  'active',
  'active',
  'active',
  'active',
  'banned',
  'suspended',
  'deleted',
  'inactive',
];
const roles: UserRole[] = ['user', 'user', 'user', 'user', 'admin'];

export const mockAdminUsers: AdminUser[] = userNames.map((name, i) => ({
  id: `admin-u-${i + 1}`,
  username: name,
  avatarUrl: avatarFor(name),
  role: roles[i % roles.length],
  status: statuses[i % statuses.length],
  joinedAt: dateOffset(rand(1, 365)),
  gamesPlayed: rand(0, 400),
  email: i % 3 === 0 ? `${name.toLowerCase()}@example.com` : undefined,
}));

export const mockAdminReports: AdminReport[] = [
  {
    id: 'r1',
    reporter: 'Alice',
    target: 'BannedUser',
    reason: 'Spamming chat',
    status: 'pending',
    createdAt: dateOffset(0),
  },
  {
    id: 'r2',
    reporter: 'Charlie',
    target: 'TrollGuy',
    reason: 'Inappropriate drawing',
    status: 'pending',
    createdAt: dateOffset(1),
  },
  {
    id: 'r3',
    reporter: 'Dina',
    target: 'RudePlayer',
    reason: 'Bad language',
    status: 'confirmed',
    createdAt: dateOffset(2),
  },
  {
    id: 'r4',
    reporter: 'Babak',
    target: 'Alice',
    reason: 'AFK too much',
    status: 'rejected',
    createdAt: dateOffset(3),
  },
  {
    id: 'r5',
    reporter: 'Faezeh',
    target: 'Golnaz',
    reason: 'Wrong word?',
    status: 'pending',
    createdAt: dateOffset(0),
  },
];

export const mockAdminCategories: AdminCategory[] = [
  { id: 'cat-animals', nameEn: 'Animals', nameFa: 'حیوانات', wordCount: 42 },
  { id: 'cat-food', nameEn: 'Food', nameFa: 'غذاها', wordCount: 35 },
  { id: 'cat-travel', nameEn: 'Travel', nameFa: 'سفر', wordCount: 28 },
  { id: 'cat-objects', nameEn: 'Objects', nameFa: 'اشیاء', wordCount: 60 },
  { id: 'cat-sports', nameEn: 'Sports', nameFa: 'ورزش', wordCount: 22 },
];

export const mockAdminWords: AdminWord[] = [
  { id: 'w1', text: 'Balloon', language: 'en', categoryId: 'cat-objects', difficulty: 'easy' },
  { id: 'w2', text: 'Apple', language: 'en', categoryId: 'cat-food', difficulty: 'easy' },
  { id: 'w3', text: 'Bicycle', language: 'en', categoryId: 'cat-sports', difficulty: 'medium' },
  { id: 'w4', text: 'Elephant', language: 'en', categoryId: 'cat-animals', difficulty: 'medium' },
  { id: 'w5', text: 'بادکنک', language: 'fa', categoryId: 'cat-objects', difficulty: 'easy' },
  { id: 'w6', text: 'سیب', language: 'fa', categoryId: 'cat-food', difficulty: 'easy' },
  { id: 'w7', text: 'دوچرخه', language: 'fa', categoryId: 'cat-sports', difficulty: 'medium' },
];

export const mockAdminBadWords: AdminBadWord[] = [
  { id: 'b1', text: 'badword1', language: 'en' },
  { id: 'b2', text: 'badword2', language: 'en' },
  { id: 'b3', text: 'کلمه‌بد', language: 'fa' },
];

export const mockAdminSongs: AdminSong[] = [
  { id: 's1', title: 'Lobby Theme 1', durationSec: 124, enabled: true },
  { id: 's2', title: 'Round Start Jingle', durationSec: 6, enabled: true },
  { id: 's3', title: 'Lobby Theme 2', durationSec: 98, enabled: false },
];
