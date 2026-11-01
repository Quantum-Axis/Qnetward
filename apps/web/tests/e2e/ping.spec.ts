import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("demo user can run ICMP ping", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/icmp-ping");
  await page.locator("#probe-target").fill("127.0.0.1");
  await page.getByRole("button", { name: /^run$/i }).click();

  await expect(page.getByRole("table").locator("tbody tr").first()).toBeVisible({ timeout: 20_000 });
  await expect(page.getByRole("heading", { name: "Latency" })).toBeVisible();
});

test("demo user can run and stop ICMP continuous mode", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/icmp-ping");
  await page.locator("#probe-target").fill("127.0.0.1");
  await page.getByRole("button", { name: /options/i }).click();
  await page.locator('input[name="count"]').fill("101");
  await page.getByLabel(/continuous/i).check();

  await expect(page.locator('input[name="count"]')).toHaveAttribute("readonly", "");

  await page.getByRole("button", { name: /^run$/i }).click();
  await expect(page.getByRole("button", { name: /^stop$/i })).toBeEnabled({ timeout: 20_000 });
  await expect(page.getByRole("table").locator("tbody tr").first()).toBeVisible({ timeout: 20_000 });

  await page.getByRole("button", { name: /^stop$/i }).click();
  await expect(page.getByText(/^Stopped$/).first()).toBeVisible({ timeout: 20_000 });
});
