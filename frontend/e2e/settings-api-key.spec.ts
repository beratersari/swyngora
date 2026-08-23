import { test, expect, type Page, type Route } from '@playwright/test';

/**
 * Browser E2E for Settings → API keys.
 * Exercises the real page, real RTK Query, and real localStorage.
 * HTTP is intercepted at the network layer (preview has no Go process).
 */

const READ_SECRET = 'swy_e2eread000000000000000000000000000000';
const AUTH_TOKEN_KEY = 'swyngora.apiAuthToken';

const emptyList = { keys: [], count: 0, clientId: 'e2e-client' };

async function stubSettingsAPI(page: Page, createdSecrets: string[]) {
  // First registered = fallback. Specific routes registered after win in Playwright.
  await page.route('**/api/**', async (route: Route) => {
    const url = route.request().url();
    const method = route.request().method();
    if (url.includes('/api/v1/account/api-keys') && method === 'POST') {
      const body = route.request().postDataJSON() as { name?: string; permission?: string };
      createdSecrets.push(String(body.permission ?? 'read'));
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'key-e2e-1',
          name: body.name ?? 'e2e',
          permission: body.permission ?? 'read',
          secret: READ_SECRET,
          prefix: 'swy_e2eread',
        }),
      });
      return;
    }
    if (url.includes('/api/v1/account/api-keys') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyList),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({}),
    });
  });
}

test.describe('Settings API key session binding', () => {
  test('creating the default Read key must not become the browser session token', async ({ page }) => {
    const createdPermissions: string[] = [];
    const authedReads: string[] = [];
    await stubSettingsAPI(page, createdPermissions);

    page.on('request', (req) => {
      if (!req.url().includes('/api/v1/account/api-keys')) return;
      if (req.method() !== 'GET') return;
      authedReads.push(req.headers()['authorization'] ?? '');
    });

    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: /settings|ayarlar/i })).toBeVisible({
      timeout: 20_000,
    });
    await expect(page.getByRole('tab', { name: /api keys|api anahtar/i })).toBeVisible();

    await page.getByLabel(/^name$|^ad$/i).fill('desk-read');
    await page.getByRole('button', { name: /create key|anahtar oluştur/i }).click();

    await expect(page.getByText(READ_SECRET)).toBeVisible({ timeout: 15_000 });
    expect(createdPermissions).toEqual(['read']);

    const stored = await page.evaluate((key) => localStorage.getItem(key), AUTH_TOKEN_KEY);
    expect(
      stored,
      'creating a Read key must not write the secret into swyngora.apiAuthToken',
    ).not.toBe(READ_SECRET);

    await page.reload();
    await expect(page.getByRole('heading', { name: /settings|ayarlar/i })).toBeVisible({
      timeout: 20_000,
    });
    await page.waitForTimeout(500);
    const later = authedReads.filter(Boolean);
    for (const header of later) {
      expect(header, 'later RTK calls must not send the Read key as Bearer').not.toBe(
        `Bearer ${READ_SECRET}`,
      );
    }
  });
});
