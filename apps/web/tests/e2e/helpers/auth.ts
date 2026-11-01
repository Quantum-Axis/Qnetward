import { APIRequestContext, expect, Page } from "@playwright/test";

export type E2EUser = {
  name: string;
  email: string;
  password: string;
};

export function resolveOrigin(baseURL?: string) {
  return new URL(baseURL ?? "http://127.0.0.1:8080").origin;
}

export async function provisionVerifiedUser(request: APIRequestContext, origin: string, user?: Partial<E2EUser>) {
  const nonce = `${Date.now()}-${Math.floor(Math.random() * 1_000_000)}`;
  const candidate: E2EUser = {
    name: user?.name ?? "E2E Operator",
    email: user?.email ?? `e2e-${nonce}@example.com`,
    password: user?.password ?? "StrongPass123!",
  };

  const signupResponse = await request.post("/api/auth/signup", {
    headers: {
      Origin: origin,
    },
    data: candidate,
  });
  // A fixed email (the admin spec) already exists after the first run; repeated runs can also hit the
  // per-email signup rate limit. Either way the account is there, and a missing one fails at sign-in.
  if (signupResponse.status() === 409 || (user?.email && signupResponse.status() === 429)) {
    return candidate;
  }
  expect(signupResponse.ok()).toBeTruthy();
  const signupPayload = await signupResponse.json();
  expect(
    signupPayload.preview_code,
    "preview_code is only available when email delivery is disabled; CI writes QNETWARD_RESEND_API_KEY='' for this reason",
  ).toBeTruthy();

  const verifyResponse = await request.post("/api/auth/verify-email", {
    headers: {
      Origin: origin,
    },
    data: {
      email: candidate.email,
      code: signupPayload.preview_code,
    },
  });
  expect(verifyResponse.ok()).toBeTruthy();

  return candidate;
}

export async function loginAsVerifiedUser(page: Page, request: APIRequestContext, origin: string, user?: Partial<E2EUser>) {
  const candidate = await provisionVerifiedUser(request, origin, user);

  await loginWithCredentials(page, candidate);

  return candidate;
}

export async function loginWithCredentials(page: Page, user: Pick<E2EUser, "email" | "password">) {
  // Wait for hydration: a click before the page is interactive does nothing.
  await page.goto("/login", { waitUntil: "networkidle" });
  await page.getByLabel("Email").fill(user.email);
  await page.locator("#login-password").fill(user.password);
  await page.getByRole("button", { name: /^sign in$/i }).click();

  await expect(page).toHaveURL(/\/overview$/, { timeout: 20_000 });
  await expect(page.getByRole("heading", { name: /welcome back/i })).toBeVisible({ timeout: 20_000 });
}

export async function loginAsDemoUser(page: Page) {
  // Wait for hydration: a click before the page is interactive does nothing.
  await page.goto("/login", { waitUntil: "networkidle" });
  await page.getByRole("button", { name: /demo workspace/i }).click();
  await expect(page).toHaveURL(/\/overview$/, { timeout: 20_000 });
  await expect(page.getByRole("heading", { name: /welcome back/i })).toBeVisible({ timeout: 20_000 });
}
