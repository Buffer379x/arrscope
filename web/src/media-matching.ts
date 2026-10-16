// Dashboard groups are mutually exclusive; detailed states remain filterable.
export const matchingGroups = [
  {
    label: "Matched",
    states: ["matched", "no_tracks"],
    help: "A unique file match was found. Audio tracks may be missing; matching does not mean that every metadata attribute is known.",
  },
  {
    label: "Not found",
    states: ["missing", "missing_size"],
    help: "No identical filename and byte size was found, or the required byte size is missing. A title match in the media server alone is not sufficient.",
  },
  {
    label: "Needs review",
    states: ["ambiguous", "conflicting"],
    help: "Multiple candidate files or streams, or conflicting metadata from enabled media servers, prevent safe enrichment. See file details for the reason.",
  },
  {
    label: "Not checked",
    states: ["pending", "unconfigured"],
    help: "Matching is pending after a file change, or no media server enrichment is configured for this file.",
  },
] as const;
export function matchingCount(
  counts: Record<string, number>,
  states: readonly string[],
) {
  return states.reduce((sum, state) => sum + (counts[state] || 0), 0);
}
