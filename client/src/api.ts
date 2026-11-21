// Owns every call to the service and the retry rule.
export const API_URL: string = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const RETRY_DELAYS_MS = [1000, 2000, 4000];

export type ErrorField = 'url' | 'code' | 'form';

export class ApiError extends Error {
  constructor(public code: string, message: string, public status?: number) {
    super(message);
  }
}

// One table: error code -> message and the field it belongs under.
export const ERRORS: Record<string, { field: ErrorField; message: string }> = {
  INVALID_URL: { field: 'url', message: 'Enter a valid http or https URL (max 2048 characters).' },
  SELF_REFERENCE: { field: 'url', message: 'That URL points to this shortener.' },
  INVALID_CODE: { field: 'code', message: 'Code must be 6–10 characters: a-z, 0-9, - or _.' },
  RESERVED_CODE: { field: 'code', message: 'That code is reserved. Choose another.' },
  CODE_TAKEN: { field: 'code', message: 'Code taken. Choose another.' },
  RATE_LIMITED: { field: 'form', message: 'Too many links created. Wait a minute and try again.' },
  UNAVAILABLE: { field: 'form', message: 'Service unavailable. Please try again later.' },
  NETWORK: { field: 'form', message: 'Cannot reach the service. Please try again later.' },
  UNKNOWN: { field: 'form', message: 'Something went wrong. Please try again.' },
};

export function describeError(err: unknown): { field: ErrorField; message: string } {
  const code = err instanceof ApiError ? err.code : 'UNKNOWN';
  return ERRORS[code] ?? ERRORS.UNKNOWN;
}

const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

async function attempt(url: string, code: string): Promise<{ shortUrl: string }> {
  let res: Response;
  try {
    res = await fetch(`${API_URL}/links`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url, code }),
    });
  } catch {
    throw new ApiError('NETWORK', 'network failure');
  }
  const body = await res.json().catch(() => null);
  if (res.ok && body && typeof body.shortUrl === 'string') return { shortUrl: body.shortUrl };
  const e = body?.error;
  throw new ApiError(e?.code ?? (res.status === 503 ? 'UNAVAILABLE' : 'UNKNOWN'), e?.message ?? res.statusText, res.status);
}

const isRetryable = (e: unknown) =>
  e instanceof ApiError && (e.code === 'NETWORK' || e.status === 503);

export async function createLink(url: string, code: string): Promise<{ shortUrl: string }> {
  for (let i = 0; ; i++) {
    try {
      return await attempt(url, code);
    } catch (e) {
      if (!isRetryable(e) || i >= RETRY_DELAYS_MS.length) throw e;
      await sleep(RETRY_DELAYS_MS[i]);
    }
  }
}
