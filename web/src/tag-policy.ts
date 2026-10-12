import type { Connection, Service, TagGroups } from "./types";
export const tagFields: (keyof TagGroups)[] = [
  "resolution",
  "codec",
  "source",
  "hdr",
  "bitDepth",
  "audio",
  "channels",
  "language",
  "custom",
];
export type TagTransition = {
  service: Service;
  enabled: string[];
  disabled: string[];
  needsMode: boolean;
};
export function tagTransition(
  service: Service,
  before: Connection,
  after: Connection,
): TagTransition {
  const enabled = tagFields.filter(
    (field) => after.tagSync?.[field] && !before.tagSync?.[field],
  );
  const disabled = tagFields.filter(
    (field) => !after.tagSync?.[field] && before.tagSync?.[field],
  );
  const active = tagFields.some((field) => after.tagSync?.[field]);
  return {
    service,
    enabled,
    disabled,
    needsMode:
      enabled.length > 0 ||
      (active &&
        (before.url !== after.url ||
          (before.tagSyncMode || "replace") !==
            (after.tagSyncMode || "replace"))),
  };
}
export type TagDecision = {
  mode?: "add" | "replace";
  disable?: "keep" | "delete";
};
