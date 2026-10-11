import { useEffect, useRef } from "preact/hooks";
import { api } from "../api";
import type { FilmSort, Results, Service } from "../types";
type QueryArgs = {
  view: string;
  service: Service;
  filters: Record<Service, string>;
  page: number;
  movieSort: FilmSort;
  lastSuccess: string | null | undefined;
  seasonMode: string | undefined;
  setResults: (r: Results) => void;
  setBusy: (b: boolean) => void;
  setError: (s: string) => void;
};
export function useLibraryQuery({
  view,
  service,
  filters,
  page,
  movieSort,
  lastSuccess,
  seasonMode,
  setResults,
  setBusy,
  setError,
}: QueryArgs) {
  const lastRequest = useRef(0);
  useEffect(() => {
    if (view !== "library") return;
    const id = ++lastRequest.current;
    const controller = new AbortController();
    setBusy(true);
    setError("");
    const timer = setTimeout(() => {
      const params = new URLSearchParams(filters[service]);
      params.set("service", service);
      params.set("page", String(page));
      if (service === "radarr") {
        params.set("sort", movieSort.field);
        params.set("dir", movieSort.direction);
      }
      api<Results>(
        "files?" + params.toString(),
        "GET",
        undefined,
        controller.signal,
      )
        .then((r) => {
          if (id === lastRequest.current) setResults(r);
        })
        .catch((e) => {
          if (e.name !== "AbortError" && id === lastRequest.current)
            setError(e.message);
        })
        .finally(() => {
          if (id === lastRequest.current) setBusy(false);
        });
    }, 180);
    return () => {
      clearTimeout(timer);
      controller.abort();
      ++lastRequest.current;
    };
  }, [
    view,
    movieSort.field,
    movieSort.direction,
    service,
    filters[service],
    page,
    lastSuccess,
    seasonMode,
  ]);
}
export function useFacets(
  enabled: boolean,
  service: Service,
  lastSuccess: string | null | undefined,
  setFacets: (f: Record<string, string[]>) => void,
  setError: (s: string) => void,
) {
  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController();
    let current = true;
    setFacets({});
    api<Record<string, string[]>>(
      "facets?service=" + service,
      "GET",
      undefined,
      controller.signal,
    )
      .then((f) => {
        if (current) setFacets(f);
      })
      .catch((e) => {
        if (current && e.name !== "AbortError") setError(e.message);
      });
    return () => {
      current = false;
      controller.abort();
    };
  }, [enabled, service, lastSuccess]);
}
