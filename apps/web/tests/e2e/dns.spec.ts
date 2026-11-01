import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("dns inspector runs a lookup and renders record sections", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/dns-lookup");

  const lookupResponsePromise = page.waitForResponse((response) =>
    response.url().includes("/api/dnslookup") && response.request().method() === "POST",
  );

  await page.locator("#dns-target").fill("example.com");
  await page.locator("#dns-record-type").selectOption("mx");
  await page.locator("#dns-resolver").selectOption("google");
  await page.getByRole("button", { name: /look up/i }).click();

  const lookupResponse = await lookupResponsePromise;
  expect(lookupResponse.ok()).toBeTruthy();

  await expect(page.getByText("8.8.8.8:53", { exact: true }).first()).toBeVisible();
  await expect(page.getByRole("heading", { name: /^MX/ })).toBeVisible();
  await expect(page.getByText("example.com", { exact: false }).first()).toBeVisible();
});
