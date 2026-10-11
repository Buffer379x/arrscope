import { getLocale, t } from "./i18n";
let csrf = "";
export async function api<T>(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const res = await fetch("/api/" + path, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
      "Accept-Language": getLocale(),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  const data = await res.json();
  if (!res.ok) throw new Error(t(data.error || "Request failed"));
  return data;
}

export function setCSRF(value: string) {
  csrf = value;
}
