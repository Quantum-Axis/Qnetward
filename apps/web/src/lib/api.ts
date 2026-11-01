import { QueryClient } from "@tanstack/react-query";

export class ApiError extends Error {
  status: number;
  payload?: unknown;

  constructor(message: string, status: number, payload?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

async function parseResponse(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return response.json();
  }
  return response.text();
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    credentials: "include",
    cache: "no-store",
  });

  return readResult<T>(response);
}

/** Sends a file as multipart form data. The browser sets the boundary, so no Content-Type here. */
export async function apiUpload<T>(path: string, field: string, file: File): Promise<T> {
  const body = new FormData();
  body.append(field, file);
  const response = await fetch(path, { method: "POST", body, credentials: "include", cache: "no-store" });
  return readResult<T>(response);
}

async function readResult<T>(response: Response): Promise<T> {
  const payload = await parseResponse(response);
  if (!response.ok) {
    const message = typeof payload === "object" && payload && "error" in payload
      ? String((payload as { error?: string }).error || "Request failed")
      : typeof payload === "string" && payload.length > 0
        ? payload
        : "Request failed";
    throw new ApiError(message, response.status, payload);
  }

  return payload as T;
}

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
        staleTime: 15_000,
      },
    },
  });
}
