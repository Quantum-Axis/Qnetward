import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("demo user can run ICMP ping", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/icmp-ping");
  await page.getByPlaceholder("example.com or 8.8.8.8").fill("127.0.0.1");
  await page.getByRole("button", { name: /run icmp ping/i }).click();

  await expect(page.getByText("Awaiting telemetry results.")).not.toBeVisible({ timeout: 20_000 });
  await expect(page.locator("tbody tr").first()).toBeVisible();
  await expect(page.getByText("Latency (ms)")).toBeVisible();
});

test("demo user can run and stop ICMP continuous mode", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/icmp-ping");
  await page.getByPlaceholder("example.com or 8.8.8.8").fill("127.0.0.1");
  await page.locator('input[name="count"]').fill("101");
  await page.getByLabel(/continuous mode/i).check();

  const countInput = page.locator('input[name="count"]');
  await expect(countInput).toHaveAttribute("readonly", "");

  await page.getByRole("button", { name: /run icmp ping/i }).click();
  await expect(page.getByText(/^Continuous$/)).toBeVisible();
  await expect(page.getByText("Awaiting telemetry results.")).not.toBeVisible({ timeout: 20_000 });
  await expect(page.getByRole("button", { name: /^Stop$/i })).toBeEnabled({ timeout: 20_000 });

  await page.getByRole("button", { name: /^Stop$/i }).click();
  await expect(page.getByText(/^Stopped$/).first()).toBeVisible({ timeout: 20_000 });
});
