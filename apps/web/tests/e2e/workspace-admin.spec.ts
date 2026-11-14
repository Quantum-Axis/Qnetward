import { expect, test, type Page } from "@playwright/test";
import { loginAsDemoUser, loginWithCredentials, provisionVerifiedUser, resolveOrigin } from "./helpers/auth";

test("admin user can access admin routes and demo user is gated out", async ({ browser, page, request, baseURL }) => {
  const origin = resolveOrigin(baseURL);
  const adminUser = await provisionVerifiedUser(request, origin, {
    name: "E2E Admin",
    email: "e2e-admin@example.com",
    password: "StrongPass123!",
  });

  await loginWithCredentials(page, adminUser);
  await page.goto("/admin");
  await expect(page.getByRole("heading", { name: "Admin overview" })).toBeVisible();

  await page.goto("/admin/users");
  await expect(page.getByRole("heading", { name: "Users", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "All users" })).toBeVisible();

  await page.goto("/admin/workspaces");
  await expect(page.getByRole("heading", { name: "Workspaces", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "All workspaces" })).toBeVisible();

  await page.goto("/admin/system");
  await expect(page.getByRole("heading", { name: "System", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Dependencies" })).toBeVisible();

  const demoContext = await browser.newContext({ baseURL });
  const demoPage = await demoContext.newPage();
  await loginAsDemoUser(demoPage);
  await demoPage.goto("/admin");
  await expect(demoPage).toHaveURL(/\/overview$/);
  await expect(demoPage.getByRole("heading", { name: /welcome back/i })).toBeVisible();
  await demoContext.close();
});

async function invite(page: Page, email: string) {
  await page.getByRole("button", { name: "Invite people" }).first().click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Email addresses").fill(email);
  await dialog.getByRole("button", { name: /^send invite$/i }).click();
  await expect(dialog).toBeHidden();
}

test("workspace owner can revoke pending invites and switch active workspace", async ({ browser, page, request, baseURL }) => {
  const origin = resolveOrigin(baseURL);
  const ownerOne = await provisionVerifiedUser(request, origin, {
    name: "Switch Owner One",
    password: "StrongPass123!",
  });
  const ownerTwo = await provisionVerifiedUser(request, origin, {
    name: "Switch Owner Two",
    password: "StrongPass123!",
  });

  const ownerTwoContext = await browser.newContext({ baseURL });
  const ownerTwoPage = await ownerTwoContext.newPage();
  await loginWithCredentials(ownerTwoPage, ownerTwo);
  await ownerTwoPage.goto("/workspace/team");
  await invite(ownerTwoPage, ownerOne.email);
  await expect(ownerTwoPage.getByText(new RegExp(ownerOne.email.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))).first()).toBeVisible();
  await ownerTwoContext.close();

  await loginWithCredentials(page, ownerOne);
  await page.goto("/workspace/team");
  const revokeEmail = `revoke-${Date.now()}@example.com`;
  await invite(page, revokeEmail);
  await expect(page.getByText(revokeEmail).first()).toBeVisible();

  await page.getByRole("button", { name: `Revoke invite for ${revokeEmail}` }).click();
  await expect(page.getByText(`Revoked invitation for ${revokeEmail}.`)).toBeVisible();

  await page.goto("/settings/workspaces");
  await expect(page.getByRole("heading", { name: "Invitations" })).toBeVisible();
  const acceptResponse = page.waitForResponse((response) => response.url().includes("/api/workspace/invitations") && response.request().method() === "POST");
  await page.getByRole("button", { name: /join workspace/i }).click();
  expect((await acceptResponse).ok()).toBeTruthy();

  await page.goto("/settings/workspaces");
  const memberships = page.locator("#memberships li");
  await expect(memberships).toHaveCount(2);
  const ownerTwoRow = memberships.filter({ hasText: "Switch Owner Two Workspace" });
  await ownerTwoRow.getByRole("button", { name: /switch to/i }).click();
  await expect(ownerTwoRow.getByText("Active", { exact: true })).toBeVisible();
  await expect
    .poll(async () => {
      return page.evaluate(async () => {
        const response = await fetch("/api/workspace/team", { credentials: "include" });
        const payload = await response.json();
        return payload.workspace?.name ?? "";
      });
    })
    .toBe("Switch Owner Two Workspace");
});
