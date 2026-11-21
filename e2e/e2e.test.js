// End-to-end test: real browser -> client -> service -> Postgres, then the redirect.
// Run with the stack up (`make up`): node e2e/e2e.test.js
const { chromium } = require('playwright');
const assert = require('node:assert');

const CLIENT = process.env.CLIENT_URL || 'http://localhost:3000';
const SERVICE = process.env.SERVICE_URL || 'http://localhost:8080';

(async () => {
  const code = 'e2e' + Math.random().toString(36).slice(2, 7); // 8 chars, unique per run
  const target = 'https://example.com/e2e';
  const browser = await chromium.launch();
  try {
    const page = await browser.newPage();
    await page.goto(CLIENT);

    // Create a link through the UI.
    await page.getByLabel(/url/i).fill(target);
    await page.getByLabel(/code/i).fill(code);
    await page.getByRole('button', { name: /shorten|create|submit/i }).click();
    await page.getByText(`t/${code}`).waitFor({ timeout: 10000 });
    console.log(`PASS create via UI shows t/${code}`);

    // Same code, different URL, via the UI -> "taken" error.
    await page.getByLabel(/url/i).fill('https://other.com');
    await page.getByRole('button', { name: /shorten|create|submit/i }).click();
    await page.getByText(/taken/i).waitFor({ timeout: 10000 });
    console.log('PASS duplicate code shows taken error');

    // Redirect: service returns 301 to the stored URL (case-insensitive code).
    const res = await fetch(`${SERVICE}/${code.toUpperCase()}`, { redirect: 'manual' });
    assert.strictEqual(res.status, 301);
    assert.strictEqual(res.headers.get('location'), target);
    console.log('PASS redirect 301 to stored URL');

    console.log('E2E: all passed');
  } catch (err) {
    console.error('E2E FAILED:', err.message);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
})();
