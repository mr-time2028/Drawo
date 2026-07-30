import type { ChatMessage } from '@/types';

const now = Date.now();

export const mockChatMessages: ChatMessage[] = [
  {
    id: 'm1',
    kind: 'system',
    text: 'Bob joined the room',
    timestamp: now - 1000 * 60 * 3,
  },
  {
    id: 'm2',
    kind: 'chat',
    userId: 'u1',
    username: 'Alice',
    text: 'Hi everyone 👋',
    timestamp: now - 1000 * 60 * 2,
  },
  {
    id: 'm3',
    kind: 'chat',
    userId: 'u3',
    username: 'Charlie',
    text: 'good luck!',
    timestamp: now - 1000 * 60,
  },
  {
    id: 'm4',
    kind: 'guess',
    userId: 'u1',
    username: 'Alice',
    text: 'apple?',
    timestamp: now - 1000 * 45,
  },
  {
    id: 'm5',
    kind: 'guess',
    userId: 'u3',
    username: 'Charlie',
    text: 'ant',
    timestamp: now - 1000 * 30,
  },
  {
    id: 'm6',
    kind: 'correct',
    userId: 'u3',
    username: 'Charlie',
    text: 'Charlie guessed the word: ant (+25)',
    timestamp: now - 1000 * 28,
  },
  {
    id: 'm7',
    kind: 'system',
    text: 'Round 3 starts — you are drawing!',
    timestamp: now - 1000 * 20,
    isSelf: true,
  },
  {
    id: 'm8',
    kind: 'chat',
    userId: 'u1',
    username: 'Alice',
    text: 'ok easy one 😄',
    timestamp: now - 1000 * 10,
  },
];
