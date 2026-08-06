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
- Pumps min-return control
- Active `aria-current` on nav

These tests do **not** require a live backend; API-dependent pages may show error banners.
