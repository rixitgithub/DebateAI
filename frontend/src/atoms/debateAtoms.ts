import { atom } from 'jotai';

// WebSocket connection atom
export const wsAtom = atom<WebSocket | null>(null);

// Debate ID atom
export const debateIdAtom = atom<string | null>(null);

export interface PollInfo {
  pollId: string;
  question: string;
  options: string[];
  counts: Record<string, number>;
  voters: number;
}

// Poll state atom: pollId -> poll info
export const pollStateAtom = atom<Record<string, PollInfo>>({});

// User spectator ID atom (generated once and stored)
export const spectatorIdAtom = atom<string>(() => {
  if (typeof window === 'undefined') return '';
  const stored = localStorage.getItem('spectatorId');
  if (stored) return stored;
  
  // Generate new spectator ID
  const spectatorId = crypto.randomUUID();
  localStorage.setItem('spectatorId', spectatorId);
  return spectatorId;
});

// User spectator hash atom (computed from spectator ID)
export const spectatorHashAtom = atom<string>(() => {
  if (typeof window === 'undefined') return '';
  const stored = localStorage.getItem('spectatorHash');
  if (stored) return stored;
  
  const spectatorId =
    localStorage.getItem('spectatorId') || crypto.randomUUID();
  if (!localStorage.getItem('spectatorId')) {
    localStorage.setItem('spectatorId', spectatorId);
  }

  // Hash is computed server-side; return the ID as a placeholder.
  return spectatorId;
});

// Transcript atom (full transcript text)
export const transcriptAtom = atom<string>('');

// Questions atom (array of questions)
export const questionsAtom = atom<Array<{ qId: string; text: string; spectatorHash: string; timestamp: number }>>([]);

// Reactions atom (array of recent reactions)
export const reactionsAtom = atom<Array<{ reaction: string; spectatorHash: string; timestamp: number }>>([]);

// WebSocket connection status atom
export const wsStatusAtom = atom<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');

// Last event ID atom (for replay)
export const lastEventIdAtom = atom<string | null>(null);

// Presence atom (connected spectators count)
export const presenceAtom = atom<number>(0);

