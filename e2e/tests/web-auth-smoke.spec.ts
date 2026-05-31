import { expect, test } from '@playwright/test';

test.describe('web auth smoke', function () {
  test('login page renders without exposing sensitive values', async function ({ page }) {
    await page.goto('/login');

    await expect(page.getByRole('heading', { name: 'ログイン' })).toBeVisible();
    await expect(page.getByText('メールアドレス')).toBeVisible();
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.getByText('パスワード')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.getByText(/session|cookie|csrf|token|secret/i)).toHaveCount(0);
  });

  test('protected extension connect redirects unauthenticated users to login', async function ({ page }) {
    await page.goto('/extension/connect?user_code=ABCDE-FGHJK');

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole('heading', { name: 'ログイン' })).toBeVisible();
  });
});
