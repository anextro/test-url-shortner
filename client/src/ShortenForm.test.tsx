import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './App';
import { API, apiError, created, server } from './test/server';

type Handler = () => Response | Promise<Response>;

// Serves responses in order and records when each request arrived (fake clock).
function sequence(...responses: Handler[]) {
  const times: number[] = [];
  server.use(
    http.post(`${API}/links`, () => {
      times.push(Date.now());
      const next = responses[Math.min(times.length - 1, responses.length - 1)];
      return next();
    }),
  );
  return times;
}

const unavailable = () => apiError(503, 'UNAVAILABLE');
const networkError = () => HttpResponse.error();

async function fill(user: ReturnType<typeof userEvent.setup>, url = 'https://example.com', code = 'abc123') {
  await user.type(screen.getByLabelText('Long URL'), url);
  await user.type(screen.getByLabelText('Code'), code);
}

const flush = () => act(() => vi.advanceTimersByTimeAsync(0));
const advance = (ms: number) => act(() => vi.advanceTimersByTimeAsync(ms));

describe('ShortenForm', () => {
  let user: ReturnType<typeof userEvent.setup>;

  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'Date'] });
    user = userEvent.setup({ delay: null });
    render(<App />);
  });
  afterEach(() => vi.useRealTimers());

  it('valid submit shows the link and copy copies t/<code>', async () => {
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    expect(await screen.findByText('t/abc123')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Copy' }));
    expect(await navigator.clipboard.readText()).toBe('t/abc123');
    expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument();
  });

  it('#19: 503 x3 then 201 retries after 1s, 2s, 4s and shows the link', async () => {
    const times = sequence(unavailable, unavailable, unavailable, () => created('abc123'));
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    await flush();
    expect(times).toHaveLength(1);
    await advance(999);
    expect(times).toHaveLength(1);
    await advance(1);
    expect(times).toHaveLength(2);
    await advance(1999);
    expect(times).toHaveLength(2);
    await advance(1);
    expect(times).toHaveLength(3);
    await advance(3999);
    expect(times).toHaveLength(3);
    await advance(1);
    expect(times).toHaveLength(4);
    expect(times.slice(1).map((t, i) => t - times[i])).toEqual([1000, 2000, 4000]);
    expect(await screen.findByText('t/abc123')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('#20: 503 x4 shows an error after 3 retries', async () => {
    const times = sequence(unavailable);
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    await flush();
    await advance(1000 + 2000 + 4000);
    expect(await screen.findByRole('alert')).toHaveTextContent(/unavailable/i);
    await advance(60_000);
    expect(times).toHaveLength(4);
    expect(screen.queryByText(/^t\//)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Shorten' })).toBeEnabled();
  });

  it('#21: 409 shows "code taken" under the code field and does not retry', async () => {
    const times = sequence(() => apiError(409, 'CODE_TAKEN', 'code already used'));
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent(/code taken/i);
    expect(alert.id).toBe('code-error');
    expect(screen.getByLabelText('Code')).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByLabelText('Code').parentElement).toContainElement(alert);
    await advance(60_000);
    expect(times).toHaveLength(1);
  });

  it('shows URL errors under the URL field without retrying', async () => {
    const times = sequence(() => apiError(400, 'INVALID_URL'));
    await fill(user, 'ftp://x.com');
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    const alert = await screen.findByRole('alert');
    expect(screen.getByLabelText('Long URL').parentElement).toContainElement(alert);
    await advance(60_000);
    expect(times).toHaveLength(1);
  });

  it('double-click submit sends one request and disables submit while in flight', async () => {
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    const times = sequence(async () => { await gate; return created('abc123'); });
    await fill(user);
    const button = screen.getByRole('button', { name: 'Shorten' });
    await user.dblClick(button);
    await flush();
    expect(button).toBeDisabled();
    expect(times).toHaveLength(1);
    release();
    expect(await screen.findByText('t/abc123')).toBeInTheDocument();
    expect(times).toHaveLength(1);
    expect(screen.getByRole('button', { name: 'Shorten' })).toBeEnabled();
  });

  it('retries network failure like 503', async () => {
    const times = sequence(networkError, networkError, () => created('abc123'));
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    await flush();
    await advance(1000);
    await advance(2000);
    expect(await screen.findByText('t/abc123')).toBeInTheDocument();
    expect(times.slice(1).map((t, i) => t - times[i])).toEqual([1000, 2000]);
  });

  it('network failure x4 shows a form-level error after 3 retries', async () => {
    const times = sequence(networkError);
    await fill(user);
    await user.click(screen.getByRole('button', { name: 'Shorten' }));
    await flush();
    await advance(7000);
    expect(await screen.findByRole('alert')).toHaveTextContent(/cannot reach/i);
    await advance(60_000);
    expect(times).toHaveLength(4);
  });
});
