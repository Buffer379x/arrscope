import type { Service } from "./types";
export type DashboardFilter = {
  field: string;
  value: string;
  scope: string;
  query: string;
};
export function makeDashboardFilter(
  field: string,
  value: string,
  scope: string,
): DashboardFilter {
  const query = new URLSearchParams();
  query.set(field, value || "__unknown__");
  return { field, value, scope, query: query.toString() };
}
export function dashboardService(scope: string, current: Service): Service {
  return scope === "sonarr"
    ? "sonarr"
    : scope === "radarr"
      ? "radarr"
      : current;
}
