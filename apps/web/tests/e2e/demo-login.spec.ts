import { expect, test } from "@playwright/test";

test("contributors can enter the full dashboard through demo login", async ({ page }) => {
  await page.goto("/login", { waitUntil: "networkidle" });

  await page.getByRole("button", { name: /demo workspace/i }).click();

  // The sign-in page is also titled "Welcome back", so check the URL, not just the heading.
  await expect(page).toHaveURL(/\/overview$/, { timeout: 20_000 });
  await expect(page.getByRole("heading", { name: /welcome back/i })).toBeVisible();
  await page.goto("/alerts");
  await expect(page.getByRole("heading", { name: /alert rules/i })).toBeVisible();
});
