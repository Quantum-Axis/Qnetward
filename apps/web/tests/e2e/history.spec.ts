import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("history page loads stored ping data after a run", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/icmp-ping");
  await page.locator("#probe-target").fill("127.0.0.1");
  await page.getByRole("button", { name: /^run$/i }).click();
  await expect(page.getByRole("status").first()).toHaveText(/finished|completed|done/i, { timeout: 30_000 });

  await page.goto("/history");
  await page.locator("#history-host").fill("127.0.0.1");
  await page.getByRole("button", { name: /^load$/i }).click();

  await expect(page.getByRole("heading", { name: /latency · 127\.0\.0\.1/i })).toBeVisible();
  await expect(page.locator("table tbody tr").first()).toBeVisible();
  await expect(page.getByRole("button", { name: /json/i })).toBeEnabled();
});
