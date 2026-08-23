# E2E (Playwright)

Browser smoke tests for the product web UI.

## Run

```bash
# Install browsers once (Chromium)
npx playwright install chromium

# Build + preview + run smoke suite
npm run test:e2e

# Against an already-running dev server
PLAYWRIGHT_BASE_URL=http://127.0.0.1:5174 npm run test:e2e
```

## Scope

- Shell navigation landmarks and routes
- Alerts query prefill (`?exchange=&symbol=`)
- Pumps scan button (no min-return control in the current UI)
- Active `aria-current` on nav
- Markets page API error copy when the backend is down
- Settings → Create API key must not install a Read secret as the browser session token (`settings-api-key.spec.ts`)

These tests do **not** require a live backend; API-dependent pages may show error banners. The Settings key spec intercepts `/api` at the browser network layer.

Chromium needs OS libraries (`libatk-1.0` and friends). If `npx playwright test` cannot launch the browser, use `PLAYWRIGHT_BASE_URL` against `npm run dev` on a machine with those libraries. `tsc -b` in `npm run build` currently fails on unrelated type errors, so prefer the Vite dev server over `webServer` build+preview.
