import { expect, test } from '@playwright/test';

test.describe('admin dry-run smoke', function () {
  test('admin route is hidden behind auth for normal browser sessions', async function ({ page }) {
    await page.goto('/admin');

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByText(/raw token|prompt本文|highlight本文|signature/i)).toHaveCount(0);
  });
});
