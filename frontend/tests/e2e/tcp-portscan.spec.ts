import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("demo user can run TCP ping against the local edge", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/tcp-ping");
  await page.getByPlaceholder("example.com or 8.8.8.8").fill("127.0.0.1");
  await page.locator("div").filter({ has: page.getByText("Port", { exact: true }) }).locator("input[type='number']").first().fill("8080");
  await page.getByRole("button", { name: /run tcp ping/i }).click();

  await expect(page.getByText("Awaiting telemetry results.")).not.toBeVisible({ timeout: 20_000 });
  await expect(page.locator("table.data-table tbody tr").first()).toBeVisible();
  await expect(page.getByText("Telemetry Summary")).toBeVisible();
});

test("demo user can run and stop TCP ping in continuous mode", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/tcp-ping");
  await page.getByPlaceholder("example.com or 8.8.8.8").fill("127.0.0.1");
  await page.locator("div").filter({ has: page.getByText("Port", { exact: true }) }).locator("input[type='number']").first().fill("8080");
  await page.locator('input[name="count"]').fill("101");
  await page.getByLabel(/continuous mode/i).check();

  const countInput = page.locator('input[name="count"]');
  await expect(countInput).toHaveAttribute("readonly", "");

  await page.getByRole("button", { name: /run tcp ping/i }).click();
  await expect(page.getByText(/^Continuous$/)).toBeVisible();
  await expect(page.getByText("Awaiting telemetry results.")).not.toBeVisible({ timeout: 20_000 });
  await expect(page.getByRole("button", { name: /^Stop$/i })).toBeEnabled({ timeout: 20_000 });

  await page.getByRole("button", { name: /^Stop$/i }).click();
  await expect(page.getByText(/^Stopped$/).first()).toBeVisible({ timeout: 20_000 });
});

test("demo user can run port scan and receive streamed results", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/port-scan");
  await page.getByPlaceholder("example.com").fill("127.0.0.1");
  await page.getByRole("button", { name: /^Web$/ }).click();
  await page.getByPlaceholder("22,80,443,8080").fill("8080,65535");
  await page.getByRole("button", { name: /run port scan/i }).click();

  await expect(page.getByText("Awaiting port results.")).not.toBeVisible({ timeout: 20_000 });
  await expect(page.locator("table tbody tr").first()).toBeVisible();
  await expect(page.getByText("Exposure Summary")).toBeVisible();
  await expect(page.getByText("HTTP Alt").first()).toBeVisible();

  await page.getByLabel(/open only/i).check();
  await expect(page.locator("table tbody tr").first()).toBeVisible();
});
