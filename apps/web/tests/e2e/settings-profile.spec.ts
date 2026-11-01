import { expect, test } from "@playwright/test";
import { loginAsDemoUser } from "./helpers/auth";

test("settings and profile pages render runtime and operator details", async ({ page }) => {
  await loginAsDemoUser(page);

  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Deployment Status" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Database, Cache, Object Storage" })).toBeVisible();
  await expect(page.getByText("Redis Cache")).toBeVisible();

  await page.goto("/profile");
  await expect(page.getByRole("heading", { name: "Profile", level: 1 })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Personal info" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Email verification" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Password" })).toBeVisible();
  await expect(page.getByText("Contributor Demo Workspace").first()).toBeVisible();
  await expect(page.getByText("demo@qnetward.local").first()).toBeVisible();
  await expect(page.getByLabel("Display Name")).toBeVisible();
  await expect(page.getByLabel("Profile Email")).toBeVisible();
  await expect(page.getByRole("button", { name: /save changes/i })).toBeDisabled();
  await expect(page.getByRole("button", { name: /send reset email/i })).toBeVisible();
});
