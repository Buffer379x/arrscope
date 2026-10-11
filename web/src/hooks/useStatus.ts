import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { api, setCSRF } from "../api";
import type { Status } from "../types";

export function useStatus() {
  const [status, setStatus] = useState<Status | null>(null);
  const [error, setError] = useState("");
  const current = useRef<Status | null>(null);
  const controller = useRef<AbortController | null>(null);
  const sequence = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout>>();
  const mounted = useRef(false);
  const schedule = useRef<() => void>(() => {});
  const refresh = useCallback(async () => {
    clearTimeout(timer.current);
    controller.current?.abort();
    const request = new AbortController();
    controller.current = request;
    const id = ++sequence.current;
    try {
      const value = await api<Status>(
        "status",
        "GET",
        undefined,
        request.signal,
      );
      if (!mounted.current || id !== sequence.current) return;
      setCSRF(value.csrf || "");
      current.current = value;
      setStatus(value);
      setError("");
    } catch (e) {
      if (
        mounted.current &&
        id === sequence.current &&
        (e as Error).name !== "AbortError"
      )
        setError((e as Error).message);
      throw e;
    } finally {
      if (id === sequence.current) schedule.current();
    }
  }, []);

  useEffect(() => {
    mounted.current = true;
    const poll = () => {
      if (!mounted.current || document.hidden) return;
      void refresh().catch(() => {});
    };
    schedule.current = () => {
      clearTimeout(timer.current);
      if (!mounted.current || document.hidden) return;
      const active = Object.values(current.current?.sync || {}).some(
        (s) => s.running,
      );
      timer.current = setTimeout(
        poll,
        active ? 3000 : current.current?.authenticated ? 30000 : 60000,
      );
    };
    const visibility = () => {
      clearTimeout(timer.current);
      if (document.hidden) controller.current?.abort();
      else poll();
    };
    poll();
    document.addEventListener("visibilitychange", visibility);
    return () => {
      mounted.current = false;
      ++sequence.current;
      clearTimeout(timer.current);
      controller.current?.abort();
      schedule.current = () => {};
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [refresh]);
  return { status, error, refresh };
}
