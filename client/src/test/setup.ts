import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, vi } from 'vitest';
import { server } from './server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => { server.resetHandlers(); cleanup(); });
afterAll(() => server.close());

// Let Testing Library's async utils (findBy*, waitFor) drive Vitest fake timers;
// it only auto-detects fake timers via a global `jest`.
(globalThis as any).jest = { advanceTimersByTime: (ms: number) => vi.advanceTimersByTime(ms) };
