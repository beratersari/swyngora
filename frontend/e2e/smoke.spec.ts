import { test, expect } from '@playwright/test';

/**
 * Smoke E2E: shell navigation + key routes render without crashing.
 * Does not require a live backend (pages may show API errors).
 */
test.describe('Swyngora web shell', () => {
  test('main navigation is labeled and links work', async ({ page }) => {
    await page.goto('/');

    const nav = page.getByRole('navigation', { name: /main|ana menü/i });
    await expect(nav).toBeVisible();

    await expect(page.getByRole('link', { name: /markets|piyasalar/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /watchlist|izleme/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /pumps|pump/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /alerts|uyarılar/i }).first()).toBeVisible();
    await expect(page.getByRole('link', { name: /compare|karşılaştır/i }).first()).toBeVisible();
  });

  test('alerts route accepts exchange/symbol prefill query', async ({ page }) => {
    await page.goto('/alerts?exchange=binance&symbol=BTCUSDT');
    // SymbolSuggest is an AutoComplete (combobox).
    await expect(page.getByRole('combobox', { name: /symbol|sembol/i })).toHaveValue('BTCUSDT', {
      timeout: 15_000,
    });
    await expect(page.getByRole('button', { name: /create alert|uyarı oluştur/i })).toBeVisible();
  });

  test('pumps page exposes min return control', async ({ page }) => {
    await page.goto('/pumps');
    await expect(page.getByText(/pump scanner|pump tarayıcı/i)).toBeVisible({ timeout: 15_000 });
    await expect(page.getByLabel(/min return|min getiri/i).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /scan|tara/i })).toBeVisible();
  });

  test('active nav marks current page', async ({ page }) => {
    await page.goto('/compare');
    const compare = page.getByRole('navigation').getByRole('link', { name: /compare|karşılaştır/i });
    await expect(compare).toHaveAttribute('aria-current', 'page');
  });

  test('shows API unreachable banner when backend is down', async ({ page }) => {
    await page.goto('/markets');
    // Without backend, health fails → banner (or markets error). Prefer health title when present.
    await expect(
      page.getByText(/API unreachable|API erişilemiyor|Could not reach the API|API'ye ulaşılamıyor/i).first(),
    ).toBeVisible({ timeout: 20_000 });
  });
});
