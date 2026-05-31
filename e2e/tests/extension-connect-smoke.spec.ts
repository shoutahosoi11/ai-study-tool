import { expect, test } from '@playwright/test';

test.describe('extension connect smoke', function () {
  test('invalid code route does not reveal token or pairing identifiers', async function ({ page }) {
    await page.goto('/extension/connect?user_code=invalid');

    await expect(page.getByText(/raw token|pairing_id|token_hash|signature/i)).toHaveCount(0);
  });
});
