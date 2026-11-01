import { expect, test } from "@playwright/test";
import { loginAsDemoUser, resolveOrigin } from "./helpers/auth";

test("alerts page can create and render a new rule", async ({ page, baseURL }) => {
  const ruleName = `ICMP Localhost Alert ${Date.now()}`;
  const origin = resolveOrigin(baseURL);
  await loginAsDemoUser(page);

  const createRuleResponse = await page.context().request.post("/api/alerts/rules", {
    headers: {
      Origin: origin,
      "Content-Type": "application/json",
    },
    data: {
      name: ruleName,
      protocol: "ping",
      target: "127.0.0.1",
      recipient_email: "demo@qnetward.local",
      latency_threshold_ms: 150,
      consecutive_breaches: 1,
      cooldown_minutes: 30,
      notify_on_recovery: true,
      enabled: true,
    },
  });
  expect(createRuleResponse.ok()).toBeTruthy();

  await page.goto("/alerts");
  await expect(page.getByRole("heading", { name: ruleName })).toBeVisible();
  await expect(page.getByText(/latency ≥ 150ms/i).first()).toBeVisible();
});
