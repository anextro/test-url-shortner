import { FormEvent, useRef, useState } from 'react';
import { createLink, describeError, ErrorField } from './api';

export function ShortenForm() {
  const [url, setUrl] = useState('');
  const [code, setCode] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ field: ErrorField; message: string } | null>(null);
  const [shortUrl, setShortUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const inFlight = useRef(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (inFlight.current) return;
    inFlight.current = true;
    setBusy(true);
    setError(null);
    setShortUrl(null);
    setCopied(false);
    try {
      const res = await createLink(url, code);
      setShortUrl(res.shortUrl);
    } catch (err) {
      setError(describeError(err));
    } finally {
      inFlight.current = false;
      setBusy(false);
    }
  }

  async function onCopy() {
    if (!shortUrl) return;
    await navigator.clipboard.writeText(shortUrl);
    setCopied(true);
  }

  const err = (f: ErrorField) =>
    error?.field === f ? <p role="alert" id={`${f}-error`}>{error.message}</p> : null;

  return (
    <form onSubmit={onSubmit} noValidate>
      <div>
        <label htmlFor="url">Long URL</label>
        <input id="url" value={url} onChange={(e) => setUrl(e.target.value)}
          aria-invalid={error?.field === 'url'} aria-describedby="url-error" />
        {err('url')}
      </div>
      <div>
        <label htmlFor="code">Code</label>
        <input id="code" value={code} onChange={(e) => setCode(e.target.value)}
          aria-invalid={error?.field === 'code'} aria-describedby="code-error" />
        {err('code')}
      </div>
      <button type="submit" disabled={busy}>{busy ? 'Shortening…' : 'Shorten'}</button>
      {err('form')}
      {shortUrl && (
        <div data-testid="result">
          <span>{shortUrl}</span>
          <button type="button" onClick={onCopy}>{copied ? 'Copied' : 'Copy'}</button>
        </div>
      )}
    </form>
  );
}
