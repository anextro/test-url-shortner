// MSW mock of the service, built from the interface contract in docs/SPEC.md.
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';

export const API = 'http://api.test';

export const created = (code: string) => HttpResponse.json({ shortUrl: `t/${code}` }, { status: 201 });
export const apiError = (status: number, code: string, message = code) =>
  HttpResponse.json({ error: { code, message } }, { status });

export const server = setupServer(
  http.post(`${API}/links`, async ({ request }) => {
    const { code } = (await request.json()) as { url: string; code: string };
    return created(code);
  }),
);
